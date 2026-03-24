# API 设计

## 通信架构总览

```
                                    ClawSynapse 网络
                                   (共享 NATS Server)
                                         │
┌──────────┐  REST API   ┌──────────────┐│  ┌─────────────┐
│  前端 UI  │ ──────────→ │  TrustMesh   ││  │ PM Agent    │
│ (React)  │ ←────────── │   后端 (Gin)  ├┤  │ (clawsynapsed│
└──────────┘  JSON       └──────┬───────┘│  │  + Adapter) │
                                │        │  └─────────────┘
                           MongoDB       │  ┌─────────────┐
                                         │  │ 执行 Agent   │
                          clawsynapsed ──┤  │ (clawsynapsed│
                          (Local API +   │  │  + Adapter) │
                           WebhookAdapter)  └─────────────┘
```

- **前端 ↔ 后端**：REST API（JSON），JWT 认证
- **TrustMesh ↔ Agent**：通过 ClawSynapse 网络通信
  - TrustMesh 发送：调用本地 `clawsynapsed` 的 Local API（`POST /v1/publish`）
  - TrustMesh 接收：WebhookAdapter 推送到 `POST /webhook/clawsynapse`
- TrustMesh 后端不直连 NATS，所有 Agent 通信由 `clawsynapsed` 代理

## 一、REST API（前端使用）

详细字段、请求参数与响应示例见 [REST API 详细接口文档](./api-reference.md)。

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
- `pm_agent_id` 为必填字段，由客户端显式传入；后端不做"自动挑选 PM Agent"。
- 项目绑定的 `pm_agent_id` 必须指向一个 `role=pm` 的 Agent。
- `pm_agent_id` 对应 Agent 必须属于当前用户。
- 如果 PM Agent 当前不在线（不在 ClawSynapse peers 列表中），不允许创建新对话或发送需求消息。
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
- `POST /api/v1/projects/:projectId/conversations` 创建 `Conversation` 时，必须同时写入首条 `role=user` 消息；MVP 不保留"空对话"。
- 用户发送消息后，后端先落库，再通过 `clawsynapsed` Local API 向 PM Agent 发送 `conversation.message` 消息。
- 首条 `conversation.message` 不是简单转发用户原文，而是附带一份增强后的 PM brief，至少应包含：
  - 任务和业务目的
  - 需求澄清要求：不明确先提问，需求未就绪前不要创建 `task.create`
  - 使用 TrustMesh Skill 回复 `conversation.reply` 的要求
  - 任务就绪标准和拆解/派发规则
  - 当前项目上下文和候选执行 Agent 列表（含 `role`、`status`、`capabilities`）
- 首条消息应同时保留 `user_content` 原始用户输入，避免 PM Agent 将系统提示和用户需求混淆。
- PM Agent 不调用 REST 回复；而是通过 ClawSynapse 网络向 TrustMesh 发送 `conversation.reply` 消息，后端校验 `role=pm` 后写回 `conversations.messages`，消息 `role=pm_agent`。
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
- 该 PM Agent 必须处于在线状态（在 ClawSynapse peers 列表中可见）；否则返回业务错误，例如 `PM_AGENT_OFFLINE`。

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
GET    /api/v1/agents/:id/insights    Agent 老板视角分析数据
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
- `node_id` 必填且唯一，对应该 Agent 的 ClawSynapse nodeId，用于消息路由和身份映射。
- `PATCH` 默认不允许修改 `node_id`。如后续需要迁移节点，应单独设计"换绑节点"接口。
- `PATCH /api/v1/agents/:id` 仅允许更新 `name`、`role`、`description`、`capabilities`。
- `role` 用于控制 PM 与执行 Agent 的权限边界。
- `capabilities` 用于 PM Agent 规划 Todo 时筛选候选执行 Agent。
- `GET /api/v1/agents` 和 `GET /api/v1/agents/:id` 应返回 Agent 在线状态相关字段，至少包括 `status`、`last_seen_at`，以支持前端展示。
- `GET /api/v1/agents/:id/insights` 用于 Agent 详情页左侧分析卡片，应该直接返回后端聚合结果，而不是让前端扫描项目/任务后自行重算。
- `GET /api/v1/agents/:id/insights` 当前关键口径：
  - `pending_over_24h` 统计年龄超过 24 小时且尚未闭环的工作项，包含 `pending` 与 `in_progress`
  - `oldest_pending_ms` 只看 `pending`
  - `longest_in_progress_ms` 只看 `in_progress`
  - PM Agent 暂无真实任务开始时间，`in_progress` 任务年龄暂以 `created_at` 作为起点
  - 执行型 Agent 的 `response_p50_ms / response_p90_ms` 口径为 `started_at - created_at`
  - 执行型 Agent 的 `completion_p50_ms / completion_p90_ms` 口径为 `completed_at - started_at`
- 当 Agent 已被 `projects.pm_agent_id`、`tasks.pm_agent_id` 或 `tasks.todos[].assignee_agent_id` 引用时，不允许物理删除，应返回业务错误。

## 二、ClawSynapse 消息协议（Agent 使用）

详细协议已拆分到 [message-protocol.md](./message-protocol.md)，该文档是唯一的消息类型和 payload 规范来源。

本文件只保留使用约定：

- TrustMesh 后端作为 ClawSynapse 节点，通过 `clawsynapsed` Local API 发送消息，通过 WebhookAdapter 接收消息
- 业务消息类型编码在 `type` 字段中，例如 `task.create`、`todo.assigned`
- Agent 身份由 webhook payload 的 `from` 字段（ClawSynapse nodeId）映射到平台内 Agent 记录
- PM 权限由 Agent 的 `role=pm` 判定
- Agent 在线状态由 ClawSynapse 节点发现机制维护，TrustMesh 通过 `GET /v1/peers` 同步

## 三、后端消息处理流程

### 典型流程：最小闭环

```
1. 用户通过 REST 发送需求消息。
2. 后端先校验项目绑定的 PM Agent 为 `role=pm` 且当前在线（peers 列表中可见）。
3. 校验通过后，后端写入 Conversation 首条用户消息，并通过 clawsynapsed Local API publish:
   POST /v1/publish → targetNode={pmNodeId}, type=conversation.message
4. PM Agent 可先通过 ClawSynapse 向 TrustMesh 发送 conversation.reply 回复用户，
   后端通过 WebhookAdapter 收到回复并写入 Conversation。
5. PM Agent 分析需求后，向 TrustMesh 发送 task.create。
6. 后端通过 WebhookAdapter 收到 task.create，校验 role=pm，
   并校验该 Conversation 尚未生成 Task；通过后写入唯一 Task + Todos，
   并将 Conversation 标记为 `resolved`。
7. 后端遍历 todos，逐个通过 clawsynapsed Local API:
   POST /v1/publish → targetNode={assigneeNodeId}, type=todo.assigned
8. 执行 Agent 收到通知，开始执行 Todo。
9. 执行 Agent 通过 ClawSynapse 回传 todo.progress / todo.complete / todo.fail。
10. 后端通过 WebhookAdapter 收到结果，更新 Todo 状态并聚合更新 Task 状态。
11. 当全部 Todo 完成时，Task 自动完成；如有 Todo 失败且未恢复，则 Task 标记失败。
```

说明：
- 具体消息类型、payload 结构和权限矩阵以 [message-protocol.md](./message-protocol.md) 为准。
- 本文件仅定义 REST 入口如何接入这条消息链路。
