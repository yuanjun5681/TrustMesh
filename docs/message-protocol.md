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
- 不在 subject 中放业务主键，如 `taskId`、`conversationId`
- 业务主键一律放进 payload

### 3.3 三类命名空间

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

### 4.1 Publish / Request 信封

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
- `id` 用作消息唯一标识和幂等键
- `timestamp` 为发送方生成时间

### 4.2 Reply 结构

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
- `error` 为稳定错误码或错误信息

## 五、身份与权限规则

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

### 7.3 PM 创建任务

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

### 7.4 Todo 指派通知

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

### 7.5 Todo 进度

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

### 7.6 Todo 完成

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
    "artifacts": [],
    "metadata": {
      "model": "gpt-5",
      "duration_ms": 1200
    }
  }
}
```

### 7.7 Todo 失败

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
