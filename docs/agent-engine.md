# Agent 引擎设计

## 通信模型

Agent 与平台的所有通信通过 NATS 服务中转，REST API 只服务前端：

> 具体 subject 命名、payload 结构、权限规则以 [message-protocol.md](./message-protocol.md) 为准。本文只描述工作流和状态流转。

```
┌──────────┐               ┌──────────────┐               ┌──────────┐               ┌──────────┐
│  前端 UI  │── REST API ──→│  TrustMesh   │── Sub/Pub ──→│   NATS   │←── Sub/Pub ──│  Agent A │
│ (React)  │←── JSON ──────│   后端 (Gin)  │←── Sub/Pub ──│   服务    │── Sub/Pub ──→│ (PM/执行) │
└──────────┘               │      ┌───┐   │               │          │               └──────────┘
                           │      │ DB│   │── Req/Reply ─→│          │←── Req/Reply ─┌──────────┐
                           │      └───┘   │←── Req/Reply ─│          │── Req/Reply ──→│  Agent B │
                           └──────────────┘               └──────────┘               │ (PM/执行) │
                                                                                     └──────────┘
```

**三种 NATS 交互模式：**

| 模式 | 方向 | 用途 |
|------|------|------|
| **Publish** | Agent → NATS → 后端 | 发布动作：创建任务、Todo 进度、Todo 结果等 |
| **Publish** | 后端 → NATS → Agent | 推送通知：新任务、状态变更、新消息等 |
| **Request-Reply** | Agent → NATS → 后端 → NATS → Agent | 查询数据：获取任务详情、待办列表等 |

- Agent 间不直接通信，一切经过 NATS 服务 + 后端中转，保证数据一致性

## PM Agent 工作流

### 工作流总览

```
用户(前端)          TrustMesh 后端          NATS 服务           PM Agent            执行 Agent
  │                      │                    │                   │                    │
  │── REST 发送消息 ─────→│                    │                   │                    │
  │                      │── 写入 DB           │                   │                    │
  │                      │── Pub 通知 ────────→│── 投递 ──────────→│                    │
  │                      │                    │                   │  分析需求            │
  │                      │                    │←── Pub 创建任务 ──│                    │
  │                      │←── 收到消息 ────────│                   │                    │
  │                      │── 写入 Task+Todos   │                   │                    │
  │                      │── Pub Todo 通知 ────→│── 投递 ────────────────────────────→│
  │                      │                    │                   │                    │
  │                      │                    │                   │                    │  执行 Todo
  │                      │                    │←── Pub 进度/结果 ─────────────────────│
  │                      │←── 收到消息 ────────│                   │                    │
  │                      │── 更新 Todo/Task     │                   │                    │
  │                      │── Pub 状态通知 ─────→│── 投递 ──────────→│                    │
  │                      │                    │                   │  可选汇总/回复       │
  │←── REST 查看状态 ────│                    │                   │                    │
  │                      │                    │                   │                    │
```

关键点：
- 用户只与后端 REST API 交互
- Agent 只与 NATS 服务交互（Pub/Sub/Request）
- 后端既是 REST 服务端，也是 NATS 的发布者和订阅者
- NATS 服务负责消息路由和投递，是 Agent 与后端之间的通信总线
- PM Agent 负责规划任务，执行 Agent 只负责接收 Todo 并回传结果
- Task 状态由后端根据 Todo 状态自动聚合
- Agent 通过周期性心跳维持在线状态，PM Agent 在线是用户提需求的前置条件

### 在线判定与 PM 门禁

- Agent 在线状态和心跳 subject 以 [message-protocol.md](./message-protocol.md) 为准。
- 项目经理身份由 Agent 的 `role=pm` 判定。
- 用户发起新对话或发送需求消息前，后端必须校验项目绑定的 PM Agent 仍然在线；如果不在线，请求直接失败，不进入需求闭环。

### 对话到任务的转化示例

约束：
- 一个 `Conversation` 最终只沉淀一个 `Task`。
- PM Agent 对同一 `Conversation` 重复发布 `task.create` 时，后端必须拒绝。

```
1. PM Agent 以 `role=pm` 被项目绑定，并持续发送 `agent.{pmNodeId}.system.heartbeat`
2. 用户 REST API 发送消息：「我需要一个用户登录功能」
3. 后端先校验该项目 PM Agent 在线，再写入 Conversation → Pub `notify.{pmNodeId}.conversation.message`
4. PM Agent 收到需求，分析后发布 `agent.{pmNodeId}.task.create`
5. `task.create` 中包含一个 Task 和多个 Todo：
   - Todo 1：后端登录接口 → Agent-B
   - Todo 2：Google OAuth 接入 → Agent-B
   - Todo 3：前端登录页面 → Agent-C
   - PM Agent 可先通过 `rpc.{pmNodeId}.agent.list` 拉取 Agent 列表，并结合 `role`、`capabilities`、`status` 选择执行者
6. 后端收到 `task.create` → 校验 role=pm 和 Conversation 唯一建 Task 约束 → 写入 MongoDB
7. 后端逐个 Todo 发布 `notify.{assigneeNodeId}.todo.assigned`
8. Agent-B、Agent-C 收到通知 → 必要时调用 `rpc.{agentNodeId}.task.get` 查看上下文 → 开始执行
9. 执行过程中，Agent 按需发布 `todo.progress`
10. 完成后，Agent 发布 `todo.complete`
11. 后端更新 Todo 状态，并在全部 Todo 完成后将 Task 聚合为 `done`
12. 用户通过 REST 查看任务状态和结果
```

### PM Agent 的特殊权限

- PM Agent 的 NATS 权限矩阵以 [message-protocol.md](./message-protocol.md) 为准。
- 普通 Agent 不允许执行 PM 专属动作。

## Agent 执行流程

### 通知驱动模式

```
Agent 启动:
  1. 连接 NATS 服务（使用注册时获得的凭证，以 nodeId 为节点标识）
  2. 周期性发布 agent.{myNodeId}.system.heartbeat
  3. 通过 NATS 订阅 notify.{myNodeId}.>（通配所有通知）
  4. 通过 NATS Request rpc.{myNodeId}.todo.assigned 检查是否有遗留 Todo

收到 NATS 通知 (todo.assigned):
  1. 通过 NATS Request rpc.{myNodeId}.task.get 获取任务详情（后端从 DB 查询后 Reply）
  2. 开始执行 Todo
  3. 执行过程中按需发布 agent.{myNodeId}.todo.progress
  4. 成功时发布 agent.{myNodeId}.todo.complete
  5. 失败时发布 agent.{myNodeId}.todo.fail
```

说明：
- 具体 subject 和 payload 结构以 [message-protocol.md](./message-protocol.md) 为准。
- 本节只定义 Agent 的行为顺序，不重复定义协议表。

### Agent 生命周期

```
创建 → 离线 → 在线(连接NATS服务+订阅) → 忙碌(执行Todo) → 在线(等待NATS通知)
```

### Todo 状态机

```
pending ──start──→ in_progress ──complete──→ done
                       │
                       └──fail──→ failed
```

### Task 状态聚合

后端不要求执行 Agent claim 整个 Task，而是根据 Todos 自动聚合 Task 状态：

```text
全部 todos = pending          → task.status = pending
存在 todo = in_progress       → task.status = in_progress
全部 todos = done             → task.status = done
存在 todo = failed            → task.status = failed
```

### 并发控制（Todo 原子更新）

后端从 NATS 收到 Todo 消息后，使用 MongoDB 原子更新单个 Todo：

```javascript
db.tasks.findOneAndUpdate(
  { _id: taskId, "todos.id": todoId, "todos.assignee_node_id": nodeId },
  {
    $set: {
      "todos.$.status": "in_progress",
      "todos.$.started_at": now,
      updated_at: now
    },
    $inc: { version: 1 }
  }
)
// 完成/失败时同理更新 todos.$ 对应字段，然后重新聚合 task.status
```

### Agent 执行结果

- `todo.complete.result` 的协议结构以 [message-protocol.md](./message-protocol.md) 为准。
- 服务端保存 Todo 级执行结果，并聚合生成任务级摘要结果。
- Todo 结果中的交付物以引用形式保存；面向用户展示的最终交付物统一收敛到 Task 级管理。

## 后端 NATS 处理层

### 架构

```go
// internal/nats/handler.go — 订阅 NATS 消息处理（Agent → NATS → 后端）
// 从主题 agent.{nodeId}.* 中提取 nodeId → 查询 Agent 记录 → 校验权限
type Handler struct {
    conn                *nats.Conn
    taskService         *service.TaskService
    conversationService *service.ConversationService
    agentService        *service.AgentService   // nodeId → Agent 映射 + role 校验
    publisher           *Publisher
}

func (h *Handler) RegisterSubscriptions()
func (h *Handler) handleTaskCreate(msg *nats.Msg)        // nodeId → Agent → 校验 role=pm
func (h *Handler) handleConversationReply(msg *nats.Msg)  // nodeId → Agent → 校验 role=pm
func (h *Handler) handleAgentHeartbeat(msg *nats.Msg)
func (h *Handler) handleTodoProgress(msg *nats.Msg)
func (h *Handler) handleTodoComplete(msg *nats.Msg)
func (h *Handler) handleTodoFail(msg *nats.Msg)

// internal/nats/rpc.go — NATS Request-Reply 处理（Agent → NATS → 后端 → NATS → Agent）
func (h *Handler) handleRPCTaskGet(msg *nats.Msg)
func (h *Handler) handleRPCTodoAssigned(msg *nats.Msg)
func (h *Handler) handleRPCProjectSummary(msg *nats.Msg)
func (h *Handler) handleRPCTaskByConversation(msg *nats.Msg)
func (h *Handler) handleRPCAgentList(msg *nats.Msg)

// internal/nats/publisher.go — 通过 NATS 发布通知（后端 → NATS → Agent）
type Publisher struct {
    conn *nats.Conn
}

func (p *Publisher) NotifyTaskCreated(nodeID, taskID string) error
func (p *Publisher) NotifyTaskUpdated(nodeID, taskID, event string) error
func (p *Publisher) NotifyConversationMessage(nodeID, conversationID, content string) error
func (p *Publisher) NotifyTodoAssigned(nodeID, taskID, todoID string) error
func (p *Publisher) NotifyTodoUpdated(nodeID, taskID, todoID, event string) error
```

说明：
- handler 绑定哪些 subject pattern，以 [message-protocol.md](./message-protocol.md) 为准。
- 这里仅保留模块职责和代码组织建议。

### 典型协作全流程

```
用户(前端)          TrustMesh 后端              NATS 服务           Agent
  │                      │                       │                  │
  │                      │                       │                  │
1.│── REST 发送对话 ─────→│                       │                  │
  │                      │── 写入 DB              │                  │
  │                      │── Pub(notify) ────────→│── 投递 ─────────→│ PM Agent
  │                      │                       │                  │
2.│                      │                       │←── Pub(conv.reply)│ PM Agent
  │                      │←── 收到消息 ───────────│                  │
  │                      │── 校验role + 写入 DB   │                  │
  │                      │                       │                  │
3.│                      │                       │←── Pub(task.create)│ PM Agent
  │                      │←── 收到消息 ───────────│                  │
  │                      │── 校验role + 写入 DB   │                  │
  │                      │── Pub(todo.assigned)──→│── 投递 ─────────→│ 执行 Agent
  │                      │                       │                  │
4.│                      │                       │←── Req(rpc.{nodeId}.task.get)│ 执行 Agent
  │                      │←── 收到请求 ───────────│                  │
  │                      │── 查 DB + Reply ──────→│── 投递 ─────────→│ 执行 Agent
  │                      │                       │                  │  执行工作...
  │                      │                       │                  │
5.│                      │                       │←── Pub(todo.complete)│ 执行 Agent
  │                      │←── 收到消息 ───────────│                  │
  │                      │── 更新 Todo/Task        │                  │
  │                      │── Pub(updated) ───────→│── 投递 ─────────→│ PM Agent
  │                      │                       │                  │
6.│                      │                       │←── Pub(conv.reply)│ PM Agent
  │                      │←── 收到消息 ───────────│                  │
  │                      │── 写入 DB              │                  │
  │←── 轮询/WS 获取汇报 ──│                       │                  │
  │                      │                       │                  │
```
