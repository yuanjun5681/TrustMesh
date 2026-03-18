# 消息传输规范

本文定义 TrustMesh 基于 ClawSynapse 网络的消息传输协议，作为 Agent 端和 TrustMesh 后端的统一实现依据。

本文件关注：
- TrustMesh 如何通过 ClawSynapse 网络收发消息
- 业务消息类型定义
- Payload 格式规范
- 消息方向与职责边界
- 身份映射与权限规则
- 核心链路完整性

本文件不关注：
- REST API 细节
- MongoDB 数据模型细节
- 前端页面交互细节
- ClawSynapse 底层协议（subject、签名、发现等），详见 [ClawSynapse 文档](https://github.com/yuanjun5681/clawsynapse)

## 一、设计目标

1. TrustMesh 后端作为 ClawSynapse 网络中的一个节点，通过 `clawsynapsed` Local API 发送消息，通过 WebhookAdapter 接收消息。
2. PM Agent 和执行 Agent 各自是独立的 ClawSynapse 节点，通过各自的 `clawsynapsed` 与网络通信。
3. 业务消息类型编码在 `type` 字段中，业务数据编码在 `message` 字段中（JSON 字符串）。
4. TrustMesh 后端仍是唯一的业务真相源，所有状态变更都先落 TrustMesh，再由 TrustMesh 通知其他 Agent。
5. Agent 间不直接通信，一切通过 TrustMesh 中转。
6. MVP 阶段使用 `TRUST_MODE=open`，不校验签名和信任关系。

## 二、通信架构

### 整体拓扑

```text
ClawSynapse 网络 (共享 NATS Server, TRUST_MODE=open)
│
├── TrustMesh 节点 (clawsynapsed + WebhookAdapter)
│   ├── 发送：POST /v1/publish (clawsynapsed Local API, 默认 127.0.0.1:18080)
│   ├── 接收：WebhookAdapter → POST {WEBHOOK_URL}
│   └── TrustMesh 后端 (Go/Gin + MongoDB)
│
├── PM Agent 节点 (clawsynapsed + OpenClawAdapter/其他)
│   └── Agent 产品 (role=pm)
│
├── 执行 Agent 节点 A (clawsynapsed + OpenClawAdapter/其他)
│   └── Agent 产品 (执行者)
│
└── 执行 Agent 节点 B ...
```

### TrustMesh 后端视角

- **发送消息**：调用本地 `clawsynapsed` 的 `POST /v1/publish`
  ```json
  {
    "targetNode": "pm-node-001",
    "type": "conversation.message",
    "message": "{\"conversation_id\":\"conv_123\",\"project_id\":\"proj_123\",\"content\":\"我需要一个登录功能\"}",
    "metadata": {}
  }
  ```
- **接收消息**：`clawsynapsed` 通过 WebhookAdapter 向 TrustMesh 的 webhook 端点发送 `POST` 请求
  ```json
  {
    "nodeId": "trustmesh-server",
    "type": "task.create",
    "from": "pm-node-001",
    "sessionKey": "",
    "message": "{\"project_id\":\"proj_123\",\"conversation_id\":\"conv_123\",\"title\":\"实现用户登录\",...}",
    "metadata": {}
  }
  ```
### Agent 端视角

- Agent 通过本地 `clawsynapsed` 的 Local API 或 Skill 发送消息
- Agent 通过本地 `clawsynapsed` 的 Adapter（OpenClawAdapter 等）接收消息
- Agent 不直接接触 NATS，也不需要了解 ClawSynapse 底层协议

## 三、消息格式

### 发送格式（POST /v1/publish）

TrustMesh 发送消息时，调用 `clawsynapsed` 的 `POST /v1/publish`：

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `targetNode` | string | 是 | 目标节点的 ClawSynapse nodeId |
| `type` | string | 是 | 业务消息类型，如 `conversation.message`、`todo.assigned` |
| `message` | string | 是 | 业务 payload，JSON 字符串 |
| `sessionKey` | string | 否 | 会话标识 |
| `metadata` | object | 否 | 附加元数据 |

成功响应：
```json
{
  "ok": true,
  "code": "msg.published",
  "message": "message published",
  "data": {
    "targetNode": "pm-node-001",
    "messageId": "msg-abc123"
  },
  "ts": 1710000000000
}
```

### 接收格式（Webhook Payload）

TrustMesh 通过 WebhookAdapter 接收消息，请求体格式：

| 字段 | 类型 | 说明 |
|------|------|------|
| `nodeId` | string | 本地节点 ID（即 TrustMesh 的 nodeId） |
| `type` | string | 业务消息类型 |
| `from` | string | 发送方节点 ID |
| `sessionKey` | string | 会话标识，可能为空 |
| `message` | string | 业务 payload，JSON 字符串 |
| `metadata` | object | 附加元数据，可能为空 |

Webhook 响应约定：
- **2xx**：投递成功
- **非 2xx**：投递失败

## 四、消息类型一览

### Agent → TrustMesh

| type | 说明 | 发送方 | 权限 |
|------|------|------|------|
| `conversation.reply` | 回复用户对话 | PM Agent | `role=pm` |
| `task.create` | 创建 Task 和 Todo 列表 | PM Agent | `role=pm` |
| `todo.progress` | Todo 执行进度 | 执行 Agent | 已指派 Agent |
| `todo.complete` | Todo 执行完成 | 执行 Agent | 已指派 Agent |
| `todo.fail` | Todo 执行失败 | 执行 Agent | 已指派 Agent |

### TrustMesh → Agent

| type | 说明 | 接收方 |
|------|------|------|
| `conversation.message` | 用户发来需求消息 | PM Agent |
| `task.created` | PM 创建任务已被服务器确认 | PM Agent |
| `task.status_changed` | Task 状态变化 | PM Agent |
| `todo.assigned` | Todo 已分配 | 执行 Agent |
| `todo.status_changed` | Todo 状态变化 | PM Agent、受影响相关 Agent |

## 五、身份与权限规则

### 身份映射

- TrustMesh 平台内每个 Agent 记录都对应一个 ClawSynapse nodeId（即 `agents.node_id`）
- TrustMesh 收到 webhook 消息后，从 `from` 字段提取 nodeId
- 用 `nodeId → Agent` 做身份映射，查找对应的平台 Agent 记录

### PM 身份判定

- PM 身份唯一由 Agent 记录中的 `role=pm` 判定
- 只有 `role=pm` 的 Agent 才有权：
  - 创建任务
  - 回复用户对话

### 在线状态判定

- Agent 在线状态由 ClawSynapse 的节点发现机制（`discovery.announce`）维护
- TrustMesh 后端通过 `GET /v1/peers` 查询已发现的节点列表
- `GET /v1/peers` 返回的 Peer 结构包含 `lastSeenMs`（最后心跳时间，Unix 毫秒）
- TrustMesh 后端应定期同步 peers 列表，更新 MongoDB 中 Agent 的 `status` 和 `last_seen_at`

`GET /v1/peers` 响应示例：
```json
{
  "ok": true,
  "code": "peers.ok",
  "data": {
    "items": [
      {
        "nodeId": "pm-node-001",
        "agentProduct": "openclaw",
        "version": "2026.3.9",
        "capabilities": ["chat", "tools"],
        "inbox": "clawsynapse.msg.pm-node-001.inbox",
        "authStatus": "authenticated",
        "trustStatus": "trusted",
        "lastSeenMs": 1710000000000,
        "metadata": {}
      }
    ]
  },
  "ts": 1710000000000
}
```

### PM 门禁

- 项目绑定的 PM Agent 必须满足 `role=pm`
- 只有当该 PM Agent 当前在线时（peers 列表中可见），用户才允许：
  - 创建新对话
  - 发送需求消息
- 若 PM 离线，服务器直接拒绝请求，返回 `PM_AGENT_OFFLINE`

### 状态通知范围

- `task.created` 只发给 PM Agent，用于确认任务已被服务端接受并写入真相源
- `task.status_changed` 只发给 PM Agent，用于感知任务聚合状态变化
- `todo.status_changed` 默认发给 PM Agent，以及除本次操作者之外的受影响相关 Agent
- 若某次 Todo 状态变化是 assignee 自己刚刚通过 `todo.progress`、`todo.complete`、`todo.fail` 提交的，TrustMesh 默认不将同义状态事件回推给该 assignee
- 若 Todo 状态是由手动派发、重派、系统取消、管理员干预等外部动作改变，则可向受影响的执行 Agent 推送 `todo.status_changed`

## 六、Payload 规范

以下定义每种消息类型的 `message` 字段内容（JSON 字符串，解析后为 JSON 对象）。

### 6.1 用户需求消息通知

type: `conversation.message`
方向: TrustMesh → PM Agent（通过 `POST /v1/publish`）

message:

首次消息示例（`is_initial_message=true`）：

```json
{
  "conversation_id": "conv_123",
  "project_id": "proj_123",
  "content": "请使用 /tm-task-plan skill 处理本次需求。首先理解用户需求，澄清不明确之处，待需求明确后再创建任务。",
  "user_content": "我需要一个用户登录功能",
  "is_initial_message": true,
  "project": {
    "name": "TrustMesh MVP",
    "description": "multi-agent task orchestration"
  },
  "pm_brief": {
    "objective": "明确任务目标和业务目的；在需求清晰前持续澄清；仅在需求满足执行条件后拆解任务并派发给其他 Agent。",
    "must_clarify_before_task_create": true,
    "must_use_skill": "tm-task-plan"
  },
  "candidate_agents": [
    {
      "id": "agent_dev_1",
      "name": "Backend Agent",
      "node_id": "node-backend-001",
      "role": "developer",
      "status": "online",
      "capabilities": ["backend", "auth"]
    }
  ]
}
```

后续消息示例（`is_initial_message=false`）：

```json
{
  "conversation_id": "conv_123",
  "project_id": "proj_123",
  "content": "用户发送了新的消息，请使用 /tm-task-plan skill 继续处理。",
  "user_content": "登录方式只需要邮箱密码，不需要 OAuth",
  "is_initial_message": false
}
```

约定：
- `content` 是系统指令，指引 PM Agent 使用 `tm-task-plan` skill 处理需求。不包含用户原始输入。
- `user_content` 始终是用户原始输入，PM Agent 应以此为准理解需求。
- `is_initial_message=true` 表示首条需求消息，首次消息携带 `project`、`pm_brief`、`candidate_agents` 上下文。
- `pm_brief.must_use_skill` 指定 PM Agent 必须使用的 skill 名称。
- `candidate_agents` 提供当前用户下可供派发的非 PM Agent 列表；PM Agent 应结合 `role`、`status` 和 `capabilities` 做分派。

### 6.2 PM 回复用户对话

type: `conversation.reply`
方向: PM Agent → TrustMesh（通过 webhook）

message:

```json
{
  "conversation_id": "conv_123",
  "content": "我会先拆解需求并安排执行"
}
```

约束：
- 仅 `role=pm` 的 Agent 可以发送
- `conversation_id` 必须属于该 PM 绑定的项目
- 若对话已进入 `resolved`，TrustMesh 必须拒绝

### 6.3 PM 创建任务

type: `task.create`
方向: PM Agent → TrustMesh（通过 webhook）

message:

```json
{
  "project_id": "proj_123",
  "conversation_id": "conv_123",
  "title": "实现用户登录",
  "description": "支持邮箱密码和 Google OAuth",
  "todos": [
    {
      "id": "todo_1",
      "title": "实现后端登录接口",
      "description": "完成邮箱密码登录 API",
      "assignee_node_id": "node-backend-001"
    },
    {
      "id": "todo_2",
      "title": "实现前端登录页",
      "description": "完成登录页和表单交互",
      "assignee_node_id": "node-frontend-001"
    }
  ]
}
```

约束：
- 同一个 `conversation_id` 只能成功创建一个 `Task`
- 若该 `Conversation` 已存在对应 Task，TrustMesh 必须拒绝重复 `task.create`

### 6.4 Task 创建完成通知

type: `task.created`
方向: TrustMesh → PM Agent（通过 `POST /v1/publish`）

message:

```json
{
  "task_id": "task_123",
  "project_id": "proj_123",
  "conversation_id": "conv_123",
  "title": "实现用户登录"
}
```

### 6.5 Task 状态更新通知

type: `task.status_changed`
方向: TrustMesh → PM Agent（通过 `POST /v1/publish`）

message:

```json
{
  "task_id": "task_123",
  "status": "in_progress",
  "actor_node_id": "node-dev-001",
  "cause": "todo.progress",
  "version": 3
}
```

约定：
- `actor_node_id` 表示触发本次状态变化的操作者节点；若为系统动作，可为空字符串或约定的系统标识
- `cause` 表示变化来源，如 `todo.progress`、`todo.complete`、`todo.fail`、`manual_dispatch`
- `version` 是服务端任务版本号，用于去重和乱序处理

### 6.6 Todo 指派通知

type: `todo.assigned`
方向: TrustMesh → 执行 Agent（通过 `POST /v1/publish`）

message:

```json
{
  "task_id": "task_123",
  "todo_id": "todo_1",
  "title": "实现后端登录接口",
  "description": "完成邮箱密码登录 API",
  "content": "你收到了一个新的 Todo 任务。请使用 /tm-task-exec skill 执行此任务，按要求回报进度和结果。",
  "exec_brief": {
    "objective": "执行分派的 Todo 任务；及时回报进度；完成后提交结果，失败时说明原因。",
    "must_use_skill": "tm-task-exec"
  }
}
```

约定：
- `content` 是系统指令，指引执行 Agent 使用 `tm-task-exec` skill 处理任务。
- `title` 和 `description` 是 Todo 的具体任务信息，执行 Agent 应以此为准理解要做什么。
- `exec_brief.must_use_skill` 指定执行 Agent 必须使用的 skill 名称。

### 6.7 Todo 状态更新通知

type: `todo.status_changed`
方向: TrustMesh → PM Agent、受影响相关 Agent（通过 `POST /v1/publish`）

message:

```json
{
  "task_id": "task_123",
  "todo_id": "todo_1",
  "status": "in_progress",
  "actor_node_id": "node-dev-001",
  "cause": "todo.progress",
  "version": 7,
  "message": "接口已完成参数校验，开始接入 JWT"
}
```

约定：
- `todo.status_changed` 是领域事件，不是命令提交回执
- 若本次变化是 assignee 自己刚通过 `todo.progress`、`todo.complete`、`todo.fail` 提交的，TrustMesh 默认不向该 assignee 回推此事件
- `actor_node_id` 表示本次状态变化的操作者节点
- `cause` 表示变化来源，如 `todo.progress`、`todo.complete`、`todo.fail`、`manual_dispatch`、`system_cancel`
- `version` 是服务端任务版本号，用于去重和乱序处理

### 6.8 Todo 进度

type: `todo.progress`
方向: 执行 Agent → TrustMesh（通过 webhook）

message:

```json
{
  "task_id": "task_123",
  "todo_id": "todo_1",
  "message": "接口已完成参数校验，开始接入 JWT"
}
```

约定：
- TrustMesh 收到某 Todo 的第一条 `todo.progress` 时，将该 Todo 状态从 `pending` 推进为 `in_progress`

### 6.9 Todo 完成

type: `todo.complete`
方向: 执行 Agent → TrustMesh（通过 webhook）

message:

```json
{
  "task_id": "task_123",
  "todo_id": "todo_1",
  "result": {
    "summary": "登录接口已完成",
    "output": "实现了注册、登录、JWT 校验",
    "artifact_refs": [
      {
        "artifact_id": "artifact_login_api",
        "kind": "report",
        "label": "登录接口实现说明"
      }
    ],
    "metadata": {
      "model": "gpt-5",
      "duration_ms": 1200,
      "transfers": [
        {
          "transfer_id": "tf_login_api_pdf",
          "size": 2048,
          "checksum": "sha256:abc123",
          "bucket": "deliverables",
          "purpose": "todo_deliverable"
        }
      ]
    }
  }
}
```

约定：
- 如果 `artifact_refs[].kind = "file"`，推荐在 `result.metadata.transfers` 中附带对应 `transfer_id`
- TrustMesh 会把这些传输信息聚合到任务级 `artifacts`，供前端继续查询文件详情

### 6.10 Todo 失败

type: `todo.fail`
方向: 执行 Agent → TrustMesh（通过 webhook）

message:

```json
{
  "task_id": "task_123",
  "todo_id": "todo_1",
  "error": "Google OAuth 凭证缺失"
}
```

## 七、错误码

| 错误码 | 说明 |
|------|------|
| `BAD_PAYLOAD` | payload 结构不合法或缺少必填字段 |
| `VALIDATION_ERROR` | 业务参数校验失败 |
| `FORBIDDEN` | 当前 Agent 无权限执行该操作 |
| `NOT_FOUND` | 目标资源不存在，或当前 Agent 无权访问该资源 |
| `PM_AGENT_OFFLINE` | 项目绑定 PM Agent 离线 |
| `CONVERSATION_TASK_EXISTS` | 对话已经关联 Task |
| `CONVERSATION_RESOLVED` | 对话已关闭，不允许继续处理 |
| `TODO_FINALIZED` | Todo 已结束，不再接受进度更新 |
| `TODO_ALREADY_DONE` | Todo 已完成 |
| `TODO_ALREADY_FAILED` | Todo 已失败 |

## 八、投递语义与可靠性

- TrustMesh → Agent 的通知为在线推送语义（通过 `POST /v1/publish`），目标 Agent 离线时不保证补投
- TrustMesh 数据库（MongoDB）是唯一真相源
- TrustMesh 默认不向“本次命令的直接发起者”回推同义状态事件；仅在状态变化对其构成新信息时才推送
- 当前具备业务级幂等或冲突保护的动作：
  - `task.create`：由 `conversation_id` 唯一约束保证同一对话只能创建一个 Task
  - `todo.complete` / `todo.fail`：Todo 已进入终态后再次提交将被拒绝
  - `todo.progress`：语义为追加进度事件，重复发送会产生重复事件，发送方应自行去重

## 九、端到端链路

### 最小闭环

```text
1. 用户通过 REST 发送需求
2. TrustMesh 后端校验项目 PM Agent:
   - role=pm
   - peers 列表中可见（在线）
3. TrustMesh 后端调用 clawsynapsed:
   POST /v1/publish → targetNode={pmNodeId}, type=conversation.message
4. PM Agent 收到消息，分析需求，向 TrustMesh 发送:
   type=task.create → targetNode={trustmeshNodeId}
5. TrustMesh 通过 WebhookAdapter 收到 task.create
   校验该 Conversation 尚未关联 Task，然后写入唯一 Task 和 Todos
6. TrustMesh 按 Todo assignee 逐个调用:
   POST /v1/publish → targetNode={assigneeNodeId}, type=todo.assigned
7. 执行 Agent 收到通知，开始执行 Todo
8. 执行 Agent 回传进度/结果:
   type=todo.progress / todo.complete / todo.fail → targetNode={trustmeshNodeId}
9. TrustMesh 通过 WebhookAdapter 收到结果，更新 Todo 状态并聚合 Task 状态
10. TrustMesh 向 PM Agent 发送状态通知:
    POST /v1/publish → targetNode={pmNodeId}, type=task.status_changed / todo.status_changed
    若该变更来自外部动作，才按需向受影响执行 Agent 发送 todo.status_changed
11. 用户通过 REST 查看结果
```

## 十、实现建议

### TrustMesh 后端

- 实现 webhook 接收端点（如 `POST /webhook/clawsynapse`），解析 webhook payload
- 根据 `type` 字段路由到对应的 handler
- 封装 `clawsynapsed` Local API 客户端，提供 `Publish` 和 `GetPeers` 方法
- 定期调用 `GET /v1/peers` 同步 Agent 在线状态到 MongoDB

```go
// internal/clawsynapse/client.go — 调用 clawsynapsed Local API
type Client struct {
    baseURL    string      // clawsynapsed Local API 地址，默认 http://127.0.0.1:18080
    httpClient *http.Client
}

func (c *Client) Publish(targetNode, msgType, message string, metadata map[string]any) (*PublishResult, error)
func (c *Client) GetPeers() ([]Peer, error)
func (c *Client) Health() (*HealthResult, error)

// internal/clawsynapse/webhook.go — 处理 WebhookAdapter 推送
type WebhookHandler struct {
    taskService         *service.TaskService
    conversationService *service.ConversationService
    agentService        *service.AgentService
    publisher           *Client
}

// HandleWebhook 是 Gin handler，注册到 POST /webhook/clawsynapse
// 根据 type 字段路由到对应处理函数
func (h *WebhookHandler) HandleWebhook(c *gin.Context)
```

### Agent 端

- Agent 通过本地 `clawsynapsed` 的 Local API 或 Skill 发送消息
- Agent 通过本地 `clawsynapsed` 的 Adapter（如 OpenClawAdapter）接收消息
- Agent 需要知道 TrustMesh 节点的 nodeId，用于 `targetNode` 字段
- Agent 可通过 `GET /v1/peers` 发现 TrustMesh 节点（`agentProduct=trustmesh`）

### clawsynapsed 配置（TrustMesh 节点侧）

```bash
clawsynapsed \
  --node-id trustmesh-server \
  --agent-adapter webhook \
  --webhook-url http://127.0.0.1:8080/webhook/clawsynapse
```

或环境变量：

```bash
NODE_ID=trustmesh-server
AGENT_ADAPTER=webhook
WEBHOOK_URL=http://127.0.0.1:8080/webhook/clawsynapse
```
