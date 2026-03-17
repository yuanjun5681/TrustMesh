# Agent 引擎设计

## 通信模型

TrustMesh 后端和所有 Agent 都是 ClawSynapse 网络中的独立节点，通过共享 NATS Server 通信。REST API 只服务前端。

> 具体消息类型、payload 结构、权限规则以 [message-protocol.md](./message-protocol.md) 为准。本文只描述工作流和状态流转。

```
                        ClawSynapse 网络 (NATS)
                               │
┌──────────┐          ┌────────┴───────┐          ┌──────────┐
│  前端 UI  │─REST API→│  TrustMesh 节点 │          │ Agent A  │
│ (React)  │←JSON─────│  后端 (Gin)    │          │(PM/执行)  │
└──────────┘          │  ┌───┐         │          │          │
                      │  │ DB│  clawsynapsed      │clawsynapsed
                      │  └───┘  │      │          │  │       │
                      └─────────┤──────┘          └──┤───────┘
                                │                    │
                         WebhookAdapter        Adapter (OpenClaw等)
                         Local API             Local API

TrustMesh 发送: POST /v1/publish (调用本地 clawsynapsed)
TrustMesh 接收: WebhookAdapter → POST /webhook/clawsynapse
Agent 发送: 通过本地 clawsynapsed Local API 或 Skill
Agent 接收: 通过本地 clawsynapsed Adapter 投递
```

**两种通信模式：**

| 模式 | 方向 | 用途 |
|------|------|------|
| **Publish** | TrustMesh → ClawSynapse → Agent | 推送通知：新任务、状态变更、新消息等 |
| **Publish** | Agent → ClawSynapse → TrustMesh | 发布动作：创建任务、Todo 进度、Todo 结果等 |
- Agent 间不直接通信，一切经过 TrustMesh 中转，保证数据一致性

## PM Agent 工作流

### 工作流总览

```
用户(前端)          TrustMesh 后端          ClawSynapse 网络      PM Agent            执行 Agent
  │                      │                    │                   │                    │
  │── REST 发送消息 ─────→│                    │                   │                    │
  │                      │── 写入 DB           │                   │                    │
  │                      │── Local API ───────→│── 投递 ──────────→│                    │
  │                      │   publish           │  conversation.    │                    │
  │                      │                    │  message           │  分析需求            │
  │                      │                    │←── 发送 ───────────│                    │
  │                      │←── WebhookAdapter ──│   task.create     │                    │
  │                      │── 写入 Task+Todos   │                   │                    │
  │                      │── Local API ───────→│── 投递 ────────────────────────────→│
  │                      │   publish           │  todo.assigned    │                    │
  │                      │                    │                   │                    │
  │                      │                    │                   │                    │  执行 Todo
  │                      │                    │←── 发送 ─────────────────────────────│
  │                      │←── WebhookAdapter ──│   todo.complete   │                    │
  │                      │── 更新 Todo/Task     │                   │                    │
  │                      │── Local API ───────→│── 投递 ──────────→│                    │
  │                      │   publish           │  task.updated     │  可选汇总/回复       │
  │←── REST 查看状态 ────│                    │                   │                    │
  │                      │                    │                   │                    │
```

关键点：
- 用户只与后端 REST API 交互
- Agent 通过各自的 `clawsynapsed` 收发消息
- TrustMesh 后端通过本地 `clawsynapsed` 的 Local API 发送消息，通过 WebhookAdapter 接收消息
- PM Agent 是独立的 ClawSynapse 节点，负责规划任务
- 执行 Agent 也是独立的 ClawSynapse 节点，只负责接收 Todo 并回传结果
- Task 状态由后端根据 Todo 状态自动聚合
- Agent 在线状态由 ClawSynapse 节点发现机制（`discovery.announce`）维护

### 在线判定与 PM 门禁

- Agent 在线状态由 ClawSynapse 的 `discovery.announce` 心跳维护。
- TrustMesh 后端通过 `GET /v1/peers` 查询 ClawSynapse 节点列表，判定 Agent 是否在线。
- 后端应定期同步 peers 列表，更新 MongoDB 中 Agent 的 `status` 和 `last_seen_at`。
- 项目经理身份由 Agent 的 `role=pm` 判定。
- 用户发起新对话或发送需求消息前，后端必须校验项目绑定的 PM Agent 仍然在线；如果不在线，请求直接失败，不进入需求闭环。

### 对话到任务的转化示例

约束：
- 一个 `Conversation` 最终只沉淀一个 `Task`。
- PM Agent 对同一 `Conversation` 重复发布 `task.create` 时，后端必须拒绝。

```
1. PM Agent 以 `role=pm` 被项目绑定，其 ClawSynapse 节点在线（peers 列表可见）
2. 用户 REST API 发送消息：「我需要一个用户登录功能」
3. 后端校验 PM Agent 在线 → 写入 Conversation → 通过 Local API publish conversation.message 给 PM
4. PM Agent 收到需求，分析后通过 ClawSynapse 向 TrustMesh 发送 task.create
5. task.create 中包含一个 Task 和多个 Todo：
   - Todo 1：后端登录接口 → Agent-B
   - Todo 2：Google OAuth 接入 → Agent-B
   - Todo 3：前端登录页面 → Agent-C
6. 后端通过 WebhookAdapter 收到 task.create → 校验 role=pm 和 Conversation 唯一建 Task 约束 → 写入 MongoDB
7. 后端逐个 Todo 通过 Local API publish todo.assigned 给对应执行 Agent
8. Agent-B、Agent-C 收到通知 → 开始执行
9. 执行过程中，Agent 按需发送 todo.progress
10. 完成后，Agent 发送 todo.complete
11. 后端通过 WebhookAdapter 收到结果，更新 Todo 状态，并在全部 Todo 完成后将 Task 聚合为 done
12. 用户通过 REST 查看任务状态和结果
```

### PM Agent 的特殊权限

- PM Agent 的权限矩阵以 [message-protocol.md](./message-protocol.md) 为准。
- 普通 Agent 不允许执行 PM 专属动作。

## Agent 执行流程

### 通知驱动模式

```
Agent 节点启动:
  1. 本地 clawsynapsed 启动，连接 NATS，发布 discovery.announce
  2. TrustMesh 通过 GET /v1/peers 感知 Agent 上线

收到 todo.assigned 通知（通过 clawsynapsed Adapter 投递）:
  1. 开始执行 Todo
  3. 执行过程中通过 clawsynapsed publish 发送进度:
     type=todo.progress → targetNode={trustmeshNodeId}
  4. 成功时发送:
     type=todo.complete → targetNode={trustmeshNodeId}
  5. 失败时发送:
     type=todo.fail → targetNode={trustmeshNodeId}
```

说明：
- 具体消息类型和 payload 结构以 [message-protocol.md](./message-protocol.md) 为准。
- 本节只定义 Agent 的行为顺序，不重复定义协议表。

### Agent 生命周期

```
创建(平台注册) → 离线 → 在线(clawsynapsed 启动 + discovery.announce) → 忙碌(执行 Todo) → 在线(等待通知)
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

后端通过 WebhookAdapter 收到 Todo 消息后，使用 MongoDB 原子更新单个 Todo：

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

- `todo.complete` 的结果结构以 [message-protocol.md](./message-protocol.md) 为准。
- 服务端保存 Todo 级执行结果，并聚合生成任务级摘要结果。
- Todo 结果中的交付物以引用形式保存；面向用户展示的最终交付物统一收敛到 Task 级管理。

## 后端 ClawSynapse 集成层

### 架构

```go
// internal/clawsynapse/client.go — 调用 clawsynapsed Local API（TrustMesh → Agent）
// 封装 POST /v1/publish、GET /v1/peers
type Client struct {
    baseURL    string       // clawsynapsed Local API 地址，默认 http://127.0.0.1:18080
    httpClient *http.Client
}

func (c *Client) Publish(targetNode, msgType, message string, metadata map[string]any) (*PublishResult, error)
func (c *Client) GetPeers() ([]Peer, error)
func (c *Client) Health() (*HealthResult, error)

// internal/clawsynapse/webhook.go — 处理 WebhookAdapter 推送（Agent → TrustMesh）
// 从 webhook payload 的 from 字段提取 nodeId → 查询 Agent 记录 → 校验权限
type WebhookHandler struct {
    taskService         *service.TaskService
    conversationService *service.ConversationService
    agentService        *service.AgentService   // nodeId → Agent 映射 + role 校验
    publisher           *Client
}

// HandleWebhook 是 Gin handler，注册到 POST /webhook/clawsynapse
func (h *WebhookHandler) HandleWebhook(c *gin.Context)
func (h *WebhookHandler) handleTaskCreate(payload *WebhookPayload)        // from → Agent → 校验 role=pm
func (h *WebhookHandler) handleConversationReply(payload *WebhookPayload)  // from → Agent → 校验 role=pm
func (h *WebhookHandler) handleTodoProgress(payload *WebhookPayload)
func (h *WebhookHandler) handleTodoComplete(payload *WebhookPayload)
func (h *WebhookHandler) handleTodoFail(payload *WebhookPayload)

// internal/clawsynapse/sync.go — 定期同步 peers 列表到 MongoDB
type PeerSyncer struct {
    client       *Client
    agentService *service.AgentService
    interval     time.Duration
}

func (s *PeerSyncer) Start(ctx context.Context)
func (s *PeerSyncer) SyncOnce(ctx context.Context) error
```

说明：
- WebhookHandler 根据 webhook payload 的 `type` 字段路由到对应处理函数。
- 所有 handler 第一件事都是：
  - 从 webhook payload `from` 提取 nodeId
  - 加载 Agent 记录
  - 校验 role
  - 校验指派关系

### 典型协作全流程

```
用户(前端)          TrustMesh 后端              ClawSynapse 网络      Agent
  │                      │                       │                  │
  │                      │                       │                  │
1.│── REST 发送对话 ─────→│                       │                  │
  │                      │── 写入 DB              │                  │
  │                      │── Local API publish ──→│── 投递 ─────────→│ PM Agent
  │                      │   conversation.message │                  │
  │                      │                       │                  │
2.│                      │                       │←── 发送 ──────────│ PM Agent
  │                      │←── WebhookAdapter ─────│  conversation.   │
  │                      │── 校验role + 写入 DB   │  reply           │
  │                      │                       │                  │
3.│                      │                       │←── 发送 ──────────│ PM Agent
  │                      │←── WebhookAdapter ─────│  task.create     │
  │                      │── 校验role + 写入 DB   │                  │
  │                      │── Local API publish ──→│── 投递 ─────────→│ 执行 Agent
  │                      │   todo.assigned        │                  │
  │                      │                       │                  │
4.│                      │                       │                  │  执行工作...
  │                      │                       │                  │
5.│                      │                       │←── 发送 ──────────│ 执行 Agent
  │                      │←── WebhookAdapter ─────│  todo.complete   │
  │                      │── 更新 Todo/Task        │                  │
  │                      │── Local API publish ──→│── 投递 ─────────→│ PM Agent
  │                      │   task.updated         │                  │
  │                      │                       │                  │
6.│                      │                       │←── 发送 ──────────│ PM Agent
  │                      │←── WebhookAdapter ─────│  conversation.   │
  │                      │── 写入 DB              │  reply           │
  │←── 轮询/WS 获取汇报 ──│                       │                  │
  │                      │                       │                  │
```
