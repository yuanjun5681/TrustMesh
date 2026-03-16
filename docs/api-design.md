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
POST   /api/v1/projects               创建项目（显式绑定 PM Agent）
GET    /api/v1/projects               列出我的项目
GET    /api/v1/projects/:id           项目详情（含 PM Agent 信息）
PATCH  /api/v1/projects/:id           更新项目
DELETE /api/v1/projects/:id           归档项目
```

`POST /api/v1/projects` 请求体示例：

```json
{
  "name": "TrustMesh MVP",
  "description": "多 Agent 协作开发项目",
  "pm_agent_id": "65f1234567890abcde001234"
}
```

约束：
- `pm_agent_id` 为必填字段，由客户端显式传入；后端不做“自动挑选 PM Agent”。
- 项目绑定的 `pm_agent_id` 必须指向一个 `role=pm` 的 Agent。
- `pm_agent_id` 对应 Agent 必须属于当前用户。
- 如果 PM Agent 当前不在线，不允许创建新对话或发送需求消息。
- `PATCH /api/v1/projects/:id` 默认只允许更新 `name`、`description`；项目归档通过 `DELETE` 语义完成。
- `GET /api/v1/projects` 和 `GET /api/v1/projects/:id` 应返回项目绑定 PM Agent 的基础展示信息，至少包括 `pm_agent_id`、PM Agent `name`、`status`、`node_id`。

### 对话（用户 ↔ PM Agent）

```
POST   /api/v1/projects/:projectId/conversations          发起新对话并发送首条需求
GET    /api/v1/projects/:projectId/conversations          列出当前用户在项目下的对话
GET    /api/v1/conversations/:id                          对话详情（含消息历史、关联任务摘要）
POST   /api/v1/conversations/:id/messages                 在活跃对话中继续发送消息
```

`POST /api/v1/projects/:projectId/conversations` 请求体示例：

```json
{
  "content": "我需要一个用户登录功能，支持邮箱密码和 Google OAuth。"
}
```

`POST /api/v1/conversations/:id/messages` 请求体示例：

```json
{
  "content": "补充一下，还需要记住登录态和退出登录。"
}
```

约定：
- `POST /api/v1/projects/:projectId/conversations` 创建 `Conversation` 时，必须同时写入首条 `role=user` 消息；MVP 不保留“空对话”。
- 用户发送消息后，后端先落库，再通过 NATS 发布 `notify.{pmNodeId}.conversation.message` 通知 PM Agent。
- PM Agent 不调用 REST 回复；而是通过 `agent.{pmNodeId}.conversation.reply` 发送回复，后端校验 `role=pm` 后写回 `conversations.messages`，消息 `role=pm_agent`。
- 前端通过 `GET /api/v1/conversations/:id` 或列表接口轮询读取 PM 回复；MVP 不依赖 WebSocket。
- `GET /api/v1/conversations/:id` 应返回 `messages`、`status`，以及如已生成任务时对应的任务摘要；该任务关系由 `tasks.conversation_id` 反查得到。

关系约束：
- 一个 `Conversation` 最终只对应一个 `Task`。
- PM Agent 针对同一 `Conversation` 只能成功创建一次任务；重复创建应返回业务错误。
- 新创建的 `Conversation.status` 为 `active`。
- PM Agent 成功创建 Task 后，对应 `Conversation.status` 应更新为 `resolved`。
- `POST /api/v1/conversations/:id/messages` 仅允许对 `status=active` 的对话追加用户消息；若已 `resolved`，返回业务错误，例如 `CONVERSATION_RESOLVED`。
- `GET /api/v1/projects/:projectId/conversations` 只返回当前登录用户在该项目下的对话。

发送前校验：
- 项目必须已绑定 PM Agent。
- 该 PM Agent 的 `role` 必须为 `pm`。
- 该 PM Agent 必须处于在线状态；否则返回业务错误，例如 `PM_AGENT_OFFLINE`。

### 任务（用户侧只读）

```
GET    /api/v1/projects/:projectId/tasks    列出任务（支持 ?status= 筛选）
GET    /api/v1/tasks/:id                     任务详情（含 Todos、result、artifacts）
GET    /api/v1/tasks/:id/events              获取事件流
```

> MVP 中任务由 PM Agent 创建，人类用户通过 UI 查看状态和结果，不直接通过 REST 创建或拆分任务。
> `GET /api/v1/tasks/:id` 应返回任务级 `result` 与 `artifacts`，作为用户侧优先读取的数据；`todos[].result` 主要用于排查执行细节。

### Agent 管理（人类用户操作）

```
POST   /api/v1/agents                 添加 Agent（按 node_id 绑定）
GET    /api/v1/agents                 列出我的 Agent
GET    /api/v1/agents/:id             Agent 详情
PATCH  /api/v1/agents/:id             更新 Agent 信息
DELETE /api/v1/agents/:id             删除 Agent（仅未被引用时允许）
```

`POST /api/v1/agents` 请求体示例：

```json
{
  "node_id": "node-dev-001",
  "name": "Backend Agent A",
  "role": "developer",
  "description": "负责后端接口与数据库实现",
  "capabilities": ["backend", "golang", "mongodb"]
}
```

`PATCH /api/v1/agents/:id` 可更新字段：

```json
{
  "name": "Backend Agent Alpha",
  "role": "reviewer",
  "description": "负责代码评审与质量检查",
  "capabilities": ["backend", "review", "security"]
}
```

约束：
- `node_id` 必填且唯一，服务端用于把 `agent.{nodeId}.>` 主题映射到 Agent。
- `PATCH` 默认不允许修改 `node_id`。如后续需要迁移节点，应单独设计“换绑节点”接口。
- `PATCH /api/v1/agents/:id` 仅允许更新 `name`、`role`、`description`、`capabilities`。
- `role` 用于控制 PM 与执行 Agent 的权限边界。
- `capabilities` 用于 PM Agent 规划 Todo 时筛选候选执行 Agent。
- `GET /api/v1/agents` 和 `GET /api/v1/agents/:id` 应返回 Agent 在线状态相关字段，至少包括 `status`、`last_seen_at`、`heartbeat_at`，以支持前端展示节点最近心跳时间。
- 当 Agent 已被 `projects.pm_agent_id`、`tasks.pm_agent_id` 或 `tasks.todos[].assignee_agent_id` 引用时，不允许物理删除，应返回业务错误。

## 二、NATS 消息协议（Agent 使用）

详细协议已拆分到 [message-protocol.md](./message-protocol.md)，该文档是唯一的 subject 和 payload 规范来源。

本文件只保留使用约定：

- Agent 发布业务动作：`agent.{nodeId}.{domain}.{action}`
- 服务器推送通知：`notify.{nodeId}.{domain}.{action}`
- Agent 发起 RPC：`rpc.{nodeId}.{domain}.{action}`
- Agent 身份由 subject 中的 `nodeId` 映射到平台内 Agent 记录
- PM 权限由 Agent 的 `role=pm` 判定
- Agent 在线状态由 `agent.{nodeId}.system.heartbeat` 维护

## 三、后端 NATS 处理流程

### 典型流程：最小闭环

```
1. 用户通过 REST 发送需求消息。
2. 后端先校验项目绑定的 PM Agent 为 `role=pm` 且当前在线。
3. 校验通过后，后端写入 Conversation 首条用户消息，并发布 → notify.{pmNodeId}.conversation.message。
4. PM Agent 可先通过 → agent.{pmNodeId}.conversation.reply 回复用户，后端将回复写回 Conversation。
5. PM Agent 分析需求后，发布 → agent.{pmNodeId}.task.create。
6. 后端校验 role=pm，并校验该 Conversation 尚未生成 Task；通过后写入唯一 Task + Todos，并将 Conversation 标记为 `resolved`。
7. 后端遍历 todos，逐个发布 → notify.{assigneeNodeId}.todo.assigned。
8. 执行 Agent 收到通知，必要时通过 rpc.{assigneeNodeId}.task.get 拉取任务详情。
9. 执行 Agent 通过 `todo.progress` / `todo.complete` / `todo.fail` 回传执行结果。
10. 后端更新 Todo 状态，并聚合更新 Task 状态。
11. 当全部 Todo 完成时，Task 自动完成；如有 Todo 失败且未恢复，则 Task 标记失败。
```

说明：
- 具体订阅 subject、消息结构和权限矩阵以 [message-protocol.md](./message-protocol.md) 为准。
- 本文件仅定义 REST 入口如何接入这条消息链路。
