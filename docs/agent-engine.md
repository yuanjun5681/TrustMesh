# Agent 引擎设计

## 通信模型

Agent 与平台的所有通信通过 NATS 服务中转，REST API 只服务前端：

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
| **Publish** | Agent → NATS → 后端 | 发布动作：领取任务、提交结果、创建任务等 |
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
  │                      │                    │                   │  分析消息            │
  │                      │                    │←── Pub 回复 ──────│                    │
  │                      │←── 收到回复 ────────│                   │                    │
  │                      │── 写入 DB           │                   │                    │
  │←── 轮询/WS 获取回复 ──│                    │                   │                    │
  │                      │                    │                   │                    │
  │  ┄┄┄┄ 多轮沟通循环（用户 ↔ 后端 ↔ NATS ↔ PM Agent）┄┄┄┄┄┄┄┄┄│                    │
  │                      │                    │                   │                    │
  │── REST 确认需求 ─────→│                    │                   │                    │
  │                      │── Pub 通知 ────────→│── 投递 ──────────→│                    │
  │                      │                    │                   │  规划+创建任务       │
  │                      │                    │←── Pub 创建任务 ──│                    │
  │                      │←── 收到消息 ────────│                   │                    │
  │                      │── 写入 Task 到 DB   │                   │                    │
  │                      │── Pub 任务通知 ─────→│── 投递 ────────────────────────────→│
  │                      │                    │                   │                    │
  │                      │                    │                   │             领取+执行
  │                      │                    │                   │                    │
  │                      │                    │←── Pub 提交结果 ──────────────────────│
  │                      │←── 收到消息 ────────│                   │                    │
  │                      │── 更新 DB           │                   │                    │
  │                      │── Pub 完成通知 ─────→│── 投递 ──────────→│                    │
  │                      │                    │                   │  检查+汇总          │
  │                      │                    │←── Pub 汇报 ──────│                    │
  │                      │←── 收到消息 ────────│                   │                    │
  │                      │── 写入 DB           │                   │                    │
  │←── 轮询/WS 获取汇报 ──│                    │                   │                    │
  │                      │                    │                   │                    │
```

关键点：
- 用户只与后端 REST API 交互
- Agent 只与 NATS 服务交互（Pub/Sub/Request）
- 后端既是 REST 服务端，也是 NATS 的发布者和订阅者
- NATS 服务负责消息路由和投递，是 Agent 与后端之间的通信总线
- 多轮沟通是循环过程，直到需求明确

### 对话到任务的转化示例

```
1. 用户 REST API 发送消息：「我需要一个用户登录功能」
2. 后端写入 DB → Pub 到 NATS → NATS 投递给 PM Agent
3. PM Agent 收到通知，Pub 回复到 NATS → NATS 投递给后端 → 写入 DB
4. 用户轮询获取回复：「需要支持哪些登录方式？」
5. 用户 REST API 回复：「邮箱密码 + Google OAuth」
6. 后端 → NATS → PM Agent 收到，分析需求后 Pub 创建任务：
   - Pub agent.{pmNodeId}.task.create: Task 1（复杂）邮箱密码登录 → Agent-B + Todos
   - Pub agent.{pmNodeId}.task.create: Task 2（复杂）Google OAuth → Agent-B + Todos
   - Pub agent.{pmNodeId}.task.create: Task 3（简单）前端登录页面 → Agent-C
7. 后端收到 → 从 nodeId 查 Agent → 校验 role=pm → 写入 DB → Pub 到 NATS → NATS 投递给 Agent-B、Agent-C
8. Agent-B 收到通知 → Request(NATS) 获取详情 → 后端 Reply(NATS) → Agent-B 执行
9. Agent-C 收到通知 → Request(NATS) 获取详情 → 后端 Reply(NATS) → Agent-C 执行
10. Agent 完成 → Pub 到 NATS → 后端收到 → 更新 DB → Pub 到 NATS → PM Agent 收到
11. PM Agent 汇总 → Pub 回复到 NATS → 后端收到 → 写入 DB → 用户获取汇报
```

### PM Agent 的特殊权限

PM Agent 使用与普通 Agent 相同的 NATS 主题，后端根据 `role=pm` 授权以下操作：

- **创建任务**：可发布 `task.create` 消息
- **指派任务**：可发布 `task.assign` 消息
- **管理 Todo**：可发布 `todo.add` 消息
- **查看全局**：可 Request `rpc.project.summary`
- **回复用户**：可发布 `conversation.reply` 消息

> 普通 Agent 发送这些消息会被后端拒绝（权限校验失败）。

## Agent 执行流程

### 通知驱动模式

```
Agent 启动:
  1. 连接 NATS 服务（使用注册时获得的凭证，以 nodeId 为节点标识）
  2. 通过 NATS 订阅 notify.{myNodeId}.>（通配所有通知）
  3. 通过 NATS Request rpc.task.assigned 检查是否有遗留任务

收到 NATS 通知 (task.assigned / todo.assigned):
  1. 通过 NATS Request rpc.task.get 获取任务详情（后端从 DB 查询后 Reply）
  2. 通过 NATS Publish agent.{myNodeId}.task.claim 领取任务
  3. 等待 NATS 通知 notify.{myNodeId}.task.claim.result 确认

  4a. 简单任务（todos 为空）:
      - 读取 task.description 作为指令
      - 执行工作
      - 通过 NATS Publish agent.{myNodeId}.task.complete 提交结果

  4b. 复杂任务（todos 不为空）:
      - 逐个执行 Todo 项：
        · 通过 NATS Publish agent.{myNodeId}.todo.update → status: in_progress
        · 执行工作
        · 通过 NATS Publish agent.{myNodeId}.todo.update → status: done + result
      - 通过 NATS Publish agent.{myNodeId}.task.progress → 报告整体进度
      - 全部 Todo 完成后：通过 NATS Publish agent.{myNodeId}.task.complete

  5. 失败时: 通过 NATS Publish agent.{myNodeId}.task.fail
```

### Agent 生命周期

```
创建 → 离线 → 在线(连接NATS服务+订阅) → 忙碌(执行任务) → 在线(等待NATS通知)
```

### 任务状态机

```
todo ──claim──→ in_progress ──complete──→ done
                     │
                     └──fail──→ failed ──retry(人工/PM)──→ todo
```

### 并发控制（乐观锁）

后端从 NATS 收到 `task.claim` 消息后，使用 MongoDB 原子操作：

```javascript
db.tasks.findOneAndUpdate(
  { _id: taskId, status: "todo", version: currentVersion },
  { $set: { status: "in_progress", assignee_id: agentId }, $inc: { version: 1 } }
)
// 返回 null → 后端通过 NATS 通知 Agent 领取失败
```

### Agent 执行结果格式

```json
{
  "summary": "完成了代码审查，发现 3 个问题",
  "output": "详细输出内容...",
  "artifacts": [
    { "type": "text", "name": "review.md", "content": "..." },
    { "type": "url", "name": "PR Link", "url": "https://..." }
  ],
  "metadata": {
    "model": "claude-sonnet-4-20250514",
    "tokens_used": 1500,
    "duration_ms": 3200
  }
}
```

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
func (h *Handler) handleTaskClaim(msg *nats.Msg)
func (h *Handler) handleTaskProgress(msg *nats.Msg)
func (h *Handler) handleTaskComplete(msg *nats.Msg)
func (h *Handler) handleTaskFail(msg *nats.Msg)
func (h *Handler) handleTodoUpdate(msg *nats.Msg)
func (h *Handler) handleTaskCreate(msg *nats.Msg)        // nodeId → Agent → 校验 role=pm
func (h *Handler) handleTaskAssign(msg *nats.Msg)        // nodeId → Agent → 校验 role=pm
func (h *Handler) handleTodoAdd(msg *nats.Msg)           // nodeId → Agent → 校验 role=pm
func (h *Handler) handleConversationReply(msg *nats.Msg)  // nodeId → Agent → 校验 role=pm

// internal/nats/rpc.go — NATS Request-Reply 处理（Agent → NATS → 后端 → NATS → Agent）
func (h *Handler) handleRPCTaskGet(msg *nats.Msg)
func (h *Handler) handleRPCTaskAssigned(msg *nats.Msg)
func (h *Handler) handleRPCProjectSummary(msg *nats.Msg)
func (h *Handler) handleRPCAgentList(msg *nats.Msg)

// internal/nats/publisher.go — 通过 NATS 发布通知（后端 → NATS → Agent）
type Publisher struct {
    conn *nats.Conn
}

func (p *Publisher) NotifyTaskAssigned(nodeID, taskID string) error
func (p *Publisher) NotifyTaskUpdated(nodeID, taskID, event string) error
func (p *Publisher) NotifyConversationMessage(nodeID, conversationID, content string) error
func (p *Publisher) NotifyTodoAssigned(nodeID, taskID, todoID string) error
func (p *Publisher) NotifyTaskClaimResult(nodeID, taskID string, success bool, err string) error
```

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
  │                      │── Pub(assigned) ──────→│── 投递 ─────────→│ 执行 Agent
  │                      │                       │                  │
4.│                      │                       │←── Req(task.get)─│ 执行 Agent
  │                      │←── 收到请求 ───────────│                  │
  │                      │── 查 DB + Reply ──────→│── 投递 ─────────→│ 执行 Agent
  │                      │                       │                  │  执行工作...
  │                      │                       │                  │
5.│                      │                       │←── Pub(complete)─│ 执行 Agent
  │                      │←── 收到消息 ───────────│                  │
  │                      │── 更新 DB              │                  │
  │                      │── Pub(updated) ───────→│── 投递 ─────────→│ PM Agent
  │                      │                       │                  │
6.│                      │                       │←── Pub(conv.reply)│ PM Agent
  │                      │←── 收到消息 ───────────│                  │
  │                      │── 写入 DB              │                  │
  │←── 轮询/WS 获取汇报 ──│                       │                  │
  │                      │                       │                  │
```
