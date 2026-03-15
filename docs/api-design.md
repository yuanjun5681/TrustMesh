# API 设计

## 通信架构总览

```
┌──────────┐  REST API   ┌──────────────┐  NATS Pub/Sub  ┌──────────┐
│  前端 UI  │ ──────────→ │  TrustMesh   │ ←────────────→ │  Agent   │
│ (React)  │ ←────────── │   后端 (Gin)  │                │ (任意语言) │
└──────────┘  JSON       └──────┬───────┘                └──────────┘
                                │
                           MongoDB
```

- **前端 ↔ 后端**：REST API（JSON），JWT 认证
- **Agent ↔ 后端**：全部通过 NATS 消息通信，Agent 不调用 HTTP 接口
- 后端同时是 NATS 的发布者和订阅者

## 一、REST API（前端使用）

### 认证方式

`Authorization: Bearer <JWT>`

### 认证

```
POST   /api/v1/auth/register          注册
POST   /api/v1/auth/login             登录（返回 JWT）
```

### 项目

```
POST   /api/v1/projects               创建项目（自动绑定 PM Agent）
GET    /api/v1/projects               列出我的项目
GET    /api/v1/projects/:id           项目详情（含 PM Agent 信息）
PATCH  /api/v1/projects/:id           更新项目
DELETE /api/v1/projects/:id           归档项目
```

### 对话（用户 ↔ PM Agent）

```
POST   /api/v1/projects/:projectId/conversations          发起新对话
GET    /api/v1/projects/:projectId/conversations           列出对话
GET    /api/v1/conversations/:id                           对话详情（含消息历史）
POST   /api/v1/conversations/:id/messages                  发送消息（用户 → PM Agent）
```

> 用户发送消息后，后端写入 DB，然后通过 NATS 发布 `conversation.message` 通知 PM Agent。

### 任务

```
POST   /api/v1/projects/:projectId/tasks    创建任务
GET    /api/v1/projects/:projectId/tasks    列出任务（支持 ?status= 筛选）
GET    /api/v1/tasks/:id                     任务详情（含 Todos）
PATCH  /api/v1/tasks/:id                     更新任务
DELETE /api/v1/tasks/:id                     删除任务
GET    /api/v1/tasks/:id/events              获取事件流
```

### 任务 Todo

```
POST   /api/v1/tasks/:id/todos              添加 Todo 项
PATCH  /api/v1/tasks/:id/todos/:todoId       更新 Todo（状态/指派/描述）
DELETE /api/v1/tasks/:id/todos/:todoId       删除 Todo 项
```

### Agent 管理（人类用户操作）

```
POST   /api/v1/agents                 注册 Agent（返回 API Key + NATS 凭证）
GET    /api/v1/agents                 列出我的 Agent
GET    /api/v1/agents/:id             Agent 详情
PATCH  /api/v1/agents/:id             更新配置
DELETE /api/v1/agents/:id             删除 Agent
```

## 二、NATS 消息协议（Agent 使用）

Agent 只需一个 NATS 连接，通过 Publish 发送动作、通过 Subscribe 接收通知、通过 Request 请求数据。

### 认证方式

Agent 注册时获得 NATS 凭证（user/token 或 NKey），连接 NATS 时使用。后端通过 NATS 账户系统控制 Agent 的发布/订阅权限。

### 2.1 Agent 发布（Agent → NATS → 后端）

所有 Agent 使用统一的主题格式 `agent.{nodeId}.{domain}.{action}`，每个 Agent 通过其网络节点（nodeId）收发消息。后端从 nodeId 映射到 Agent 记录，根据 `role` 做权限校验。

| 主题 | 说明 | 权限 | Payload |
|------|------|------|---------|
| `agent.{nodeId}.task.claim` | 领取任务 | 所有 Agent | `{ "task_id": "..." }` |
| `agent.{nodeId}.task.progress` | 报告进度 | 所有 Agent | `{ "task_id": "...", "message": "..." }` |
| `agent.{nodeId}.task.complete` | 提交结果 | 所有 Agent | `{ "task_id": "...", "result": {...} }` |
| `agent.{nodeId}.task.fail` | 报告失败 | 所有 Agent | `{ "task_id": "...", "error": "..." }` |
| `agent.{nodeId}.todo.update` | 更新 Todo | 所有 Agent | `{ "task_id": "...", "todo_id": "...", "status": "...", "result": {...} }` |
| `agent.{nodeId}.task.create` | 创建任务 | role=pm | `{ "project_id": "...", "title": "...", "todos": [...], "assignee_node_id": "..." }` |
| `agent.{nodeId}.task.assign` | 指派任务 | role=pm | `{ "task_id": "...", "assignee_node_id": "..." }` |
| `agent.{nodeId}.todo.add` | 添加 Todo | role=pm | `{ "task_id": "...", "todos": [...] }` |
| `agent.{nodeId}.conversation.reply` | 回复用户对话 | role=pm | `{ "conversation_id": "...", "content": "..." }` |

> 权限校验在后端 handler 层实现：收到消息后从主题中提取 `nodeId`，查询对应的 Agent 记录获取 role，判断是否有权执行该操作。非 PM Agent 发送 `task.create` 等消息会被拒绝。

### 2.2 Agent 订阅（后端 → NATS → Agent）

后端处理完业务逻辑后，发布通知到目标 Agent 的 nodeId 主题，Agent 订阅接收：

| 主题 | 触发时机 | Payload |
|------|---------|---------|
| `notify.{nodeId}.task.assigned` | 任务被指派给该 Agent | `{ "task_id": "...", "project_id": "...", "title": "..." }` |
| `notify.{nodeId}.task.updated` | 关联任务状态变更 | `{ "task_id": "...", "status": "...", "event": "..." }` |
| `notify.{nodeId}.todo.assigned` | Todo 被指派给该 Agent | `{ "task_id": "...", "todo_id": "...", "title": "..." }` |
| `notify.{nodeId}.conversation.message` | 用户发来对话消息 | `{ "conversation_id": "...", "content": "..." }` |
| `notify.{nodeId}.task.claim.result` | 领取任务的结果 | `{ "task_id": "...", "success": true, "error": "..." }` |

### 2.3 Agent 请求（Request-Reply 模式）

Agent 需要读取数据时，使用 NATS Request-Reply（同步请求，带超时）。消息信封中携带 nodeId，后端据此做权限校验：

| 请求主题 | 说明 | 权限 | Request Payload | Reply Payload |
|---------|------|------|----------------|---------------|
| `rpc.task.get` | 获取任务详情 | 所有 Agent | `{ "task_id": "..." }` | `{ "task": {...} }` |
| `rpc.task.assigned` | 获取我的待办任务 | 所有 Agent | `{}` | `{ "tasks": [...] }` |
| `rpc.project.summary` | 获取项目全局状态 | role=pm | `{ "project_id": "..." }` | `{ "summary": {...} }` |
| `rpc.agent.list` | 获取可用 Agent 列表 | role=pm | `{ "owner_id": "..." }` | `{ "agents": [...] }` |

### 2.4 消息通用格式

所有 NATS 消息使用 JSON，包含统一的信封结构：

```json
{
  "id": "uuid",
  "timestamp": "2026-03-15T10:30:00Z",
  "node_id": "string",
  "payload": { ... }
}
```

Reply 消息包含成功/失败标志：

```json
{
  "success": true,
  "data": { ... },
  "error": ""
}
```

## 三、后端 NATS 处理流程

### 订阅处理示例

```
后端启动时订阅（通配 nodeId，handler 内部从主题提取 nodeId → 查 Agent → 校验 role）:
  agent.*.task.claim              → handleTaskClaim()
  agent.*.task.progress           → handleTaskProgress()
  agent.*.task.complete           → handleTaskComplete()
  agent.*.task.fail               → handleTaskFail()
  agent.*.task.create             → handleTaskCreate()        // 校验 role=pm
  agent.*.task.assign             → handleTaskAssign()        // 校验 role=pm
  agent.*.todo.update             → handleTodoUpdate()
  agent.*.todo.add                → handleTodoAdd()           // 校验 role=pm
  agent.*.conversation.reply      → handleConversationReply() // 校验 role=pm
  rpc.task.get                    → handleRPCTaskGet()
  rpc.task.assigned               → handleRPCTaskAssigned()
  rpc.project.summary             → handleRPCProjectSummary() // 校验 role=pm
  rpc.agent.list                  → handleRPCAgentList()      // 校验 role=pm
```

### 典型流程：创建并指派任务

```
1. PM Agent 发布 → agent.{pmNodeId}.task.create
2. 后端收到消息：
   a. 从主题提取 pmNodeId → 查询 Agent 记录，校验 role=pm
   b. 写入 Task 到 MongoDB
   c. 发布 → notify.{targetNodeId}.task.assigned（通知目标 Agent）
   d. 发布 → notify.{pmNodeId}.task.claim.result（确认创建成功）
3. 目标 Agent 收到通知
4. 目标 Agent 发送 Request → rpc.task.get（获取任务详情）
5. 后端 Reply 返回任务数据
6. Agent 执行任务...
```
