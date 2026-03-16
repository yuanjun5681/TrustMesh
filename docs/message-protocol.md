# 消息传输规范

本文定义 TrustMesh 在 NATS 上的消息传输协议，作为 Agent 端和服务器端的统一实现依据。

本文件关注：
- subject 命名规范
- 消息方向与职责边界
- 通用消息结构
- 权限和身份校验
- 核心链路完整性

本文件不关注：
- REST API 细节
- MongoDB 数据模型细节
- 前端页面交互细节

## 一、设计目标

1. Agent 与服务器之间只通过 NATS 通信，不走 HTTP。
2. subject 命名必须能一眼看出消息方向、主体身份、业务域和动作。
3. PM Agent 与执行 Agent 使用同一套协议，只通过 `role` 区分权限。
4. 服务器是唯一的业务真相源，所有状态变更都先落服务器，再由服务器通知其他 Agent。
5. Agent 间不直接通信，一切通过服务器中转。

## 二、角色与方向

### Agent 端视角

- 发布业务动作到 `agent.{nodeId}.*`
- 订阅服务器通知 `notify.{nodeId}.>`
- 通过 `rpc.{nodeId}.*` 发起请求查询
- 周期性发送心跳，维持在线状态

### 服务器端视角

- 订阅 `agent.*.*.*` 处理 Agent 上报动作
- 订阅 `rpc.*.*.*` 处理 Agent 查询请求
- 发布 `notify.{nodeId}.*` 向目标 Agent 推送通知
- 根据 subject 中的 `nodeId` 映射到平台内 Agent 记录

## 三、Subject 命名规范

### 3.1 总体格式

NATS subject 统一使用四段式：

```text
{namespace}.{nodeId}.{domain}.{action}
```

字段说明：
- `namespace`：消息命名空间，取值为 `agent`、`notify`、`rpc`
- `nodeId`：Agent 节点标识
- `domain`：业务域，例如 `system`、`conversation`、`task`、`todo`、`project`、`agent`
- `action`：动作名，例如 `heartbeat`、`create`、`reply`、`assigned`、`updated`、`get`

### 3.2 命名规则

- 全部使用小写字母
- 单词之间使用 `.` 分段
- 复合动作使用下划线，例如 `by_conversation`
- `nodeId` 不允许包含 `.`
- `nodeId` 推荐使用 `[a-z0-9-]+`
- 不在 subject 中放业务主键，如 `taskId`、`conversationId`
- 业务主键一律放进 payload

### 3.3 与 NATS 交互模式对应

为避免将 `rpc` 误解为另一种“发布消息”方式，这里明确协议命名空间与 NATS 原生交互模式的对应关系：

- `agent.*`：基于 NATS `publish/subscribe`，用于 Agent 向服务器提交业务动作
- `notify.*`：基于 NATS `publish/subscribe`，用于服务器向目标 Agent 推送通知
- `rpc.*`：基于 NATS `request/reply`，用于 Agent 向服务器发起查询或同步请求，并等待 reply

补充说明：
- 从协议语义看，当前主要使用两类交互模式：`publish/subscribe` 与 `request/reply`
- `queue group` 属于订阅侧的消费分发方式，不是本文当前协议暴露给业务方的独立消息类型
- 本文当前不定义基于 `queue group` 的额外 subject 约定

### 3.4 三类命名空间

#### `agent`

表示 Agent 主动向服务器发布业务动作。

格式：

```text
agent.{sourceNodeId}.{domain}.{action}
```

语义：
- 第二段 `nodeId` 表示消息发送方
- 服务器收到后，必须校验 `nodeId` 对应的 Agent 是否存在
- 权限由该 Agent 的 `role` 决定

#### `notify`

表示服务器向目标 Agent 主动推送通知。

格式：

```text
notify.{targetNodeId}.{domain}.{action}
```

语义：
- 第二段 `nodeId` 表示消息接收方
- Agent 只订阅自己的 `notify.{myNodeId}.>`

#### `rpc`

表示 Agent 发起 request-reply 查询。

格式：

```text
rpc.{callerNodeId}.{domain}.{action}
```

语义：
- 第二段 `nodeId` 表示请求发起方
- 服务器通过该 `nodeId` 做身份映射和权限校验
- reply 走 NATS inbox，不再单独定义 reply subject

## 四、通用消息结构

### 4.1 Agent Publish / RPC Request 信封

所有 `agent.*` 和 `rpc.*` 消息统一使用以下结构：

```json
{
  "id": "uuid",
  "timestamp": "2026-03-16T10:30:00Z",
  "node_id": "node-dev-001",
  "payload": {}
}
```

约束：
- `node_id` 必须与 subject 中的 `{nodeId}` 一致
- `id` 为发送方生成的消息请求 ID，同时用作链路追踪 ID
- `timestamp` 为发送方生成时间，使用 UTC RFC3339
- `payload` 必须为 JSON object；无业务参数时使用 `{}`，不使用 `null`

补充说明：
- 当前 MVP 中，`id` 已进入协议，但尚未在服务端做“所有动作统一去重”
- 发送方不得假设“同一 `id` 重发一定被后端自动忽略”
- 当前具备业务级幂等或冲突保护的动作：
  - `task.create`：由 `conversation_id` 唯一约束保证同一对话只能创建一个 Task
  - `todo.complete` / `todo.fail`：Todo 已进入终态后再次提交将被拒绝
  - `todo.progress`：语义为追加进度事件，重复发送会产生重复事件，发送方应自行去重或避免无意义重试

### 4.2 Notify 消息结构

所有 `notify.*` 消息当前直接发送业务 payload，不再包一层 envelope。

示例：

```json
{
  "task_id": "task_123",
  "todo_id": "todo_1",
  "status": "in_progress",
  "message": "接口已完成参数校验"
}
```

约束：
- `notify.*` payload 必须直接包含消费方恢复上下文所需的业务主键
- 若通知表达“状态变化”，则 payload 必须包含变化后的最新状态，而不是只发送动作名
- 当前 `notify.*` 不携带统一的 `id` / `timestamp`；需要强一致恢复时，Agent 应以数据库查询 RPC 结果为准

### 4.3 RPC Reply 结构

所有 RPC reply 使用统一结构：

```json
{
  "success": true,
  "data": {},
  "error": ""
}
```

失败时：
- `success=false`
- `error` 必须为稳定错误码，不使用临时自然语言

常见错误码：

| 错误码 | 说明 |
|------|------|
| `BAD_ENVELOPE` | 外层 envelope 非法，例如 JSON 错误、缺少 `node_id`、`node_id` 与 subject 不一致 |
| `BAD_PAYLOAD` | payload 结构不合法或缺少必填字段 |
| `UNSUPPORTED_RPC` | 服务端未实现该 RPC |
| `VALIDATION_ERROR` | 业务参数校验失败 |
| `FORBIDDEN` | 当前 Agent 无权限执行该操作 |
| `NOT_FOUND` | 目标资源不存在，或当前 Agent 无权访问该资源 |
| `PM_AGENT_OFFLINE` | 项目绑定 PM Agent 离线 |
| `CONVERSATION_TASK_EXISTS` | 对话已经关联 Task |
| `CONVERSATION_RESOLVED` | 对话已关闭，不允许继续处理 |
| `TODO_FINALIZED` | Todo 已结束，不再接受进度更新 |
| `TODO_ALREADY_DONE` | Todo 已完成 |
| `TODO_ALREADY_FAILED` | Todo 已失败 |

### 4.4 投递语义与顺序约束

- 当前协议基于 Core NATS，不依赖 JetStream 持久化
- `notify.*` 为在线推送语义，目标 Agent 离线时不保证补投
- 跨 subject 不保证全局顺序，例如 `notify.{nodeId}.task.updated` 与 `notify.{nodeId}.todo.updated` 可能先后到达
- 服务端数据库状态是唯一真相源；Agent 收到通知后如需强一致上下文，应补拉 RPC，例如 `rpc.{nodeId}.task.get`
- Agent 重启恢复的标准入口是 `rpc.{nodeId}.todo.assigned`，而不是依赖离线期间漏掉的通知重放

## 五、身份与权限规则

### 5.0 连接认证前提

- 协议默认前提：Backend 和 Agent 都通过已认证的 NATS 连接接入
- Agent 连接在 NATS 权限层只应被允许：
  - publish `agent.{selfNodeId}.>`
  - publish `rpc.{selfNodeId}.>`
  - subscribe `notify.{selfNodeId}.>`
- Backend 连接在 NATS 权限层只应被允许：
  - subscribe `agent.*.*.*`
  - subscribe `rpc.*.*.*`
  - publish `notify.*`
- NATS 连接鉴权和 subject ACL 是第一层安全边界；服务端收到消息后的 `nodeId -> Agent` 映射和 `role` 校验是第二层业务边界

### 5.1 身份映射

- 平台内每个 Agent 都有唯一 `node_id`
- 服务器收到 NATS 消息后，先从 subject 中提取 `nodeId`
- 再用 `nodeId -> Agent` 做身份映射

### 5.2 PM 身份判定

- 项目经理身份不来自 subject 名称
- PM 身份唯一由 Agent 记录中的 `role=pm` 判定
- 只有 `role=pm` 的 Agent 才有权：
  - 创建任务
  - 回复用户对话
  - 查询项目汇总
  - 查询候选执行 Agent

### 5.3 在线状态判定

- 每个 Agent 必须周期性发送 `agent.{nodeId}.system.heartbeat`
- 服务器根据最近一次心跳时间更新：
  - `heartbeat_at`
  - `last_seen_at`
  - `status`
- 超过心跳超时阈值未收到心跳，则该 Agent 进入 `offline`

### 5.4 PM 门禁

- 项目绑定的 PM Agent 必须满足 `role=pm`
- 只有当该 PM Agent 当前在线时，用户才允许：
  - 创建新对话
  - 发送需求消息
- 若 PM 离线，服务器直接拒绝请求，例如返回 `PM_AGENT_OFFLINE`

## 六、Subject 一览

### 6.1 Agent -> Server

| Subject | 说明 | 发送方 | 权限 |
|------|------|------|------|
| `agent.{nodeId}.system.heartbeat` | 心跳保活 | 所有 Agent | 所有 Agent |
| `agent.{nodeId}.conversation.reply` | 回复用户对话 | PM Agent | `role=pm` |
| `agent.{nodeId}.task.create` | 创建 Task 和 Todo 列表 | PM Agent | `role=pm` |
| `agent.{nodeId}.todo.progress` | Todo 执行进度 | 执行 Agent | 已指派 Agent |
| `agent.{nodeId}.todo.complete` | Todo 执行完成 | 执行 Agent | 已指派 Agent |
| `agent.{nodeId}.todo.fail` | Todo 执行失败 | 执行 Agent | 已指派 Agent |

### 6.2 Server -> Agent

| Subject | 说明 | 接收方 |
|------|------|------|
| `notify.{nodeId}.conversation.message` | 用户发来需求消息 | PM Agent |
| `notify.{nodeId}.task.created` | PM 创建任务已被服务器接收 | PM Agent |
| `notify.{nodeId}.task.updated` | Task 状态变化 | PM Agent |
| `notify.{nodeId}.todo.assigned` | Todo 已分配 | 执行 Agent |
| `notify.{nodeId}.todo.updated` | Todo 状态变化 | 相关 Agent |

### 6.3 RPC

| Subject | 说明 | 发起方 | 权限 |
|------|------|------|------|
| `rpc.{nodeId}.task.get` | 获取任务详情 | 所有 Agent | 所有 Agent |
| `rpc.{nodeId}.todo.assigned` | 获取当前 Agent 的待执行 Todo | 所有 Agent | 所有 Agent |
| `rpc.{nodeId}.project.summary` | 获取项目摘要 | PM Agent | `role=pm` |
| `rpc.{nodeId}.task.by_conversation` | 获取某对话对应的唯一任务 | PM Agent | `role=pm` |
| `rpc.{nodeId}.agent.list` | 获取候选执行 Agent | PM Agent | `role=pm` |

## 七、Payload 规范

### 7.1 心跳

Subject:

```text
agent.{nodeId}.system.heartbeat
```

Payload:

```json
{
  "status": "online",
  "timestamp": "2026-03-16T10:30:00Z"
}
```

说明：
- `status` 允许 `online` 或 `busy`
- `offline` 不通过心跳主动上报，而由服务器超时推导

### 7.2 用户需求消息通知

Subject:

```text
notify.{pmNodeId}.conversation.message
```

Payload:

```json
{
  "conversation_id": "conv_123",
  "project_id": "proj_123",
  "content": "我需要一个用户登录功能"
}
```

### 7.3 PM 回复用户对话

Subject:

```text
agent.{pmNodeId}.conversation.reply
```

Payload:

```json
{
  "conversation_id": "conv_123",
  "content": "我会先拆解需求并安排执行"
}
```

约束：
- 仅 `role=pm` 的 Agent 可以发送
- `conversation_id` 必须属于该 PM 绑定的项目
- 若对话已进入 `resolved`，服务器必须拒绝

### 7.4 PM 创建任务

Subject:

```text
agent.{pmNodeId}.task.create
```

Payload:

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
- 同一个 `conversation_id` 只能成功创建一个 `Task`。
- 若该 `Conversation` 已存在对应 Task，服务器必须拒绝重复 `task.create`。

### 7.5 Task 创建完成通知

Subject:

```text
notify.{pmNodeId}.task.created
```

Payload:

```json
{
  "task_id": "task_123",
  "project_id": "proj_123",
  "conversation_id": "conv_123",
  "title": "实现用户登录"
}
```

说明：
- 该消息表示服务器已经成功落库并生成 Task
- 该消息不是“创建请求回执”的唯一真相；真实状态仍应以 DB / RPC 查询结果为准

### 7.6 Task 状态更新通知

Subject:

```text
notify.{pmNodeId}.task.updated
```

Payload:

```json
{
  "task_id": "task_123",
  "status": "in_progress"
}
```

说明：
- 当前用于向 PM Agent 通知任务聚合状态变化
- `status` 当前取值为 `pending`、`in_progress`、`done`、`failed`

### 7.7 Todo 指派通知

Subject:

```text
notify.{assigneeNodeId}.todo.assigned
```

Payload:

```json
{
  "task_id": "task_123",
  "todo_id": "todo_1",
  "title": "实现后端登录接口",
  "description": "完成邮箱密码登录 API"
}
```

说明：
- 执行 Agent 如需完整上下文，应继续请求 `rpc.{nodeId}.task.get`

### 7.8 Todo 状态更新通知

Subject:

```text
notify.{nodeId}.todo.updated
```

Payload:

```json
{
  "task_id": "task_123",
  "todo_id": "todo_1",
  "status": "in_progress",
  "message": "接口已完成参数校验，开始接入 JWT"
}
```

说明：
- 接收方可能是 Todo assignee，也可能是 PM Agent
- `status` 当前取值为 `pending`、`in_progress`、`done`、`failed`
- `message` 为可选展示字段：
  - 当来源为 `todo.progress` 时，通常携带进度文本
  - 当来源为 `todo.complete` 时，当前实现可写为 `completed`
  - 当来源为 `todo.fail` 时，当前实现可写为失败原因

### 7.9 Todo 进度

Subject:

```text
agent.{nodeId}.todo.progress
```

Payload:

```json
{
  "task_id": "task_123",
  "todo_id": "todo_1",
  "message": "接口已完成参数校验，开始接入 JWT"
}
```

约定：
- 服务器收到某 Todo 的第一条 `todo.progress` 时，将该 Todo 状态从 `pending` 推进为 `in_progress`

### 7.10 Todo 完成

Subject:

```text
agent.{nodeId}.todo.complete
```

Payload:

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
      "duration_ms": 1200
    }
  }
}
```

### 7.11 Todo 失败

Subject:

```text
agent.{nodeId}.todo.fail
```

Payload:

```json
{
  "task_id": "task_123",
  "todo_id": "todo_1",
  "error": "Google OAuth 凭证缺失"
}
```

### 7.12 RPC: 获取任务详情

Subject:

```text
rpc.{nodeId}.task.get
```

Request Payload:

```json
{
  "task_id": "task_123"
}
```

Success Reply:

```json
{
  "success": true,
  "data": {
    "id": "task_123",
    "project_id": "proj_123",
    "conversation_id": "conv_123",
    "title": "实现用户登录",
    "status": "in_progress",
    "todos": []
  }
}
```

说明：
- `data` 返回完整 Task 详情对象
- 返回字段以平台 `TaskDetail` 领域模型为准

### 7.13 RPC: 获取当前 Agent 的待执行 Todo

Subject:

```text
rpc.{nodeId}.todo.assigned
```

Request Payload:

```json
{}
```

Success Reply:

```json
{
  "success": true,
  "data": {
    "items": [
      {
        "task_id": "task_123",
        "project_id": "proj_123",
        "conversation_id": "conv_123",
        "task_title": "实现用户登录",
        "task_status": "pending",
        "todo": {
          "id": "todo_1",
          "title": "实现后端登录接口",
          "status": "pending"
        }
      }
    ]
  }
}
```

说明：
- 用于 Agent 启动恢复和断线重连后的补拉
- `items` 中每一项返回一个被分配给当前 Agent 且尚未结束的 Todo

### 7.14 RPC: 获取项目摘要

Subject:

```text
rpc.{pmNodeId}.project.summary
```

Request Payload:

```json
{
  "project_id": "proj_123"
}
```

Success Reply:

```json
{
  "success": true,
  "data": {
    "id": "proj_123",
    "name": "Demo Project",
    "status": "active",
    "task_count": 3,
    "done_task_count": 1
  }
}
```

说明：
- 仅 `role=pm` 且绑定该项目的 PM Agent 可调用
- `data` 返回完整 `ProjectSummary` 结构

### 7.15 RPC: 根据对话查询唯一任务

Subject:

```text
rpc.{pmNodeId}.task.by_conversation
```

Request Payload:

```json
{
  "conversation_id": "conv_123"
}
```

Success Reply:

```json
{
  "success": true,
  "data": {
    "id": "task_123",
    "conversation_id": "conv_123",
    "status": "done"
  }
}
```

说明：
- 仅 `role=pm` 且该对话属于其绑定项目时可调用
- `data` 返回完整 Task 详情对象

### 7.16 RPC: 获取候选执行 Agent

Subject:

```text
rpc.{pmNodeId}.agent.list
```

Request Payload:

```json
{
  "project_id": "proj_123"
}
```

其中：
- `project_id` 为可选
- 若传入，则用于基于项目上下文筛选同一用户下可见的候选 Agent

Success Reply:

```json
{
  "success": true,
  "data": {
    "items": [
      {
        "id": "agent_backend_1",
        "name": "Backend Agent",
        "role": "developer",
        "node_id": "node-backend-001",
        "status": "online"
      }
    ]
  }
}
```

说明：
- 仅 `role=pm` 的 Agent 可调用
- `items` 返回候选 Agent 列表；返回字段以平台 `Agent` 领域模型为准

## 八、端到端链路

### 8.1 最小闭环

```text
1. 用户通过 REST 发送需求
2. 服务器校验项目 PM Agent:
   - role=pm
   - status=online
3. 服务器发布 notify.{pmNodeId}.conversation.message
4. PM Agent 分析需求，发布 agent.{pmNodeId}.task.create
5. 服务器校验该 Conversation 尚未关联 Task，然后写入唯一 Task 和 Todos
6. 服务器按 Todo assignee 发布 notify.{assigneeNodeId}.todo.assigned
7. 执行 Agent 拉取 rpc.{nodeId}.task.get 或直接开始执行
8. 执行 Agent 发布 todo.progress / todo.complete / todo.fail
9. 服务器更新 Todo 状态并聚合 Task 状态
10. 服务器向 PM Agent 发布 task.updated / todo.updated
11. 用户通过 REST 查看结果
```

### 8.2 Agent 启动恢复链路

```text
1. Agent 建立 NATS 连接
2. Agent 周期性发送 agent.{nodeId}.system.heartbeat
3. Agent 订阅 notify.{nodeId}.>
4. Agent 请求 rpc.{nodeId}.todo.assigned
5. 服务器返回该 Agent 当前未完成的 Todo 列表
6. Agent 恢复执行
```

## 九、完整性与自洽性审查

### 9.1 已闭合的链路

- 用户需求 -> PM 收到通知
- PM 规划 -> 创建 Task + Todos
- 服务器落库 -> 按 Todo 指派
- 执行 Agent 处理 -> 回传结果
- 服务器聚合 -> Task 状态更新
- PM 在线门禁 -> 避免需求投递到离线 PM
- 心跳保活 -> 在线状态可计算

### 9.2 当前约束下的结论

- 对最小 MVP 来说，链路已经完整，可以支撑开发
- 当前设计选择“PM 不在线则禁止发起需求”，以换取更简单的离线恢复逻辑
- 当前设计不支持 Agent 间直连，也不支持离线消息堆积，这是有意的 MVP 收敛

### 9.3 后续版本可扩展点

- 使用 JetStream 持久化关键通知
- 为 PM Agent 增加“未处理需求补拉”RPC
- 增加更细粒度的错误码和重试策略
- 为 `task.create`、`todo.complete` 增加严格幂等去重

## 十、实现建议

### Agent 端

- 封装统一 NATS 客户端
- 固定实现：
  - heartbeat sender
  - notify subscriber
  - rpc requester
  - publish helper
- 所有 subject 拼装统一走一个 subject builder

### 服务器端

- 统一实现 subject parser
- 按 namespace 分三类 handler：
  - `agent.*`
  - `notify.*` 仅 publisher 使用
  - `rpc.*`
- 所有 handler 第一件事都是：
  - 解析 subject
  - 加载 Agent
  - 校验 role
  - 校验指派关系
