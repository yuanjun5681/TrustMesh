# REST API 详细接口文档

> 本文是前端对接 TrustMesh REST API 的详细契约文档。
> 高层入口、链路和约束以 [API 设计](./api-design.md) 为准；字段、请求和响应以本文为准。

## 一、基础约定

### 1.1 Base URL

```text
/api/v1
```

### 1.2 认证

- 除注册、登录外，所有接口都需要携带 JWT：

```http
Authorization: Bearer <token>
```

### 1.3 Content-Type

```http
Content-Type: application/json
```

### 1.4 时间与 ID

- 所有时间字段使用 ISO 8601 UTC 字符串，例如 `2026-03-16T10:30:00Z`
- 资源主键 `id` 为字符串形式的 MongoDB ObjectId
- `null` 表示字段当前无值，不用空字符串占位

### 1.5 统一响应格式

成功响应：

```json
{
  "data": {}
}
```

列表响应：

```json
{
  "data": {
    "items": []
  },
  "meta": {
    "count": 0
  }
}
```

错误响应：

```json
{
  "error": {
    "code": "PM_AGENT_OFFLINE",
    "message": "项目绑定的 PM Agent 当前离线",
    "details": {}
  }
}
```

说明：

- 除 `204 No Content` 外，所有成功响应都返回 JSON body
- `204 No Content` 不返回响应体
- 列表接口统一使用 `data.items + meta.count`

### 1.6 HTTP 状态码约定

| 状态码 | 说明 |
|------|------|
| `200 OK` | 查询成功、更新成功、归档成功 |
| `201 Created` | 创建成功 |
| `204 No Content` | 删除成功且不返回响应体 |
| `400 Bad Request` | 请求格式错误 |
| `401 Unauthorized` | 未登录或 JWT 无效 |
| `403 Forbidden` | 无权访问当前资源 |
| `404 Not Found` | 资源不存在 |
| `409 Conflict` | 业务状态冲突，例如 PM 离线、对话已结束、Agent 已被引用 |
| `422 Unprocessable Entity` | 字段值通过 JSON 解析但未通过业务校验 |

### 1.7 字段与命名约定

- JSON 请求体和响应体字段统一使用 `snake_case`
- 路由占位参数在文档中使用 `:projectId`、`:taskId` 这类 lowerCamelCase，仅用于说明路径变量
- 列表响应统一放在 `data.items`，总数统一放在 `meta.count`
- 当前 MVP 列表接口都不做分页；若后续增加分页，继续沿用 `data.items + meta`
- 查询参数省略表示“不筛选”，例如 `GET /projects/:projectId/tasks` 不带 `status` 时返回全部状态

### 1.8 TypeScript 基础泛型

建议前端统一使用以下基础类型包裹接口返回值：

```ts
export interface ApiError {
  code: string
  message: string
  details: Record<string, unknown>
}

export interface ApiResponse<T> {
  data: T
}

export interface ApiListMeta {
  count: number
}

export interface ApiListResponse<T> {
  data: {
    items: T[]
  }
  meta: ApiListMeta
}
```

### 1.9 常用错误码

| 错误码 | 说明 |
|------|------|
| `UNAUTHORIZED` | JWT 缺失、无效或已过期 |
| `FORBIDDEN` | 当前用户无权访问该资源 |
| `VALIDATION_ERROR` | 请求参数校验失败 |
| `NOT_FOUND` | 资源不存在 |
| `PM_AGENT_OFFLINE` | 项目绑定的 PM Agent 当前离线 |
| `CONVERSATION_RESOLVED` | 对话已结束，不允许继续追加消息 |
| `CONVERSATION_TASK_EXISTS` | 该对话已生成任务 |
| `AGENT_NODE_ID_EXISTS` | `node_id` 已被占用 |
| `AGENT_IN_USE` | Agent 已被项目、任务或 Todo 引用，不能删除 |
| `PROJECT_PM_AGENT_INVALID` | `pm_agent_id` 不存在、不是 `role=pm` 或不属于当前用户 |
| `INVALID_CREDENTIALS` | 邮箱或密码错误 |

## 二、通用对象结构

### 2.1 User

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | `string` | 用户 ID |
| `email` | `string` | 登录邮箱 |
| `name` | `string` | 用户名称 |
| `created_at` | `string` | 创建时间 |
| `updated_at` | `string` | 更新时间 |

### 2.2 PMAgentSummary

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | `string` | Agent ID |
| `name` | `string` | Agent 名称 |
| `node_id` | `string` | 节点标识 |
| `status` | `"online" \| "offline" \| "busy"` | 在线状态 |

### 2.3 Project

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | `string` | 项目 ID |
| `name` | `string` | 项目名称 |
| `description` | `string` | 项目描述 |
| `status` | `"active" \| "archived"` | 项目状态 |
| `pm_agent` | [`PMAgentSummary`](#22-pmagentsummary) | 项目绑定的 PM Agent |
| `created_at` | `string` | 创建时间 |
| `updated_at` | `string` | 更新时间 |

### 2.4 ConversationMessage

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | `string` | 消息 ID，uuid |
| `role` | `"user" \| "pm_agent"` | 消息发送方角色 |
| `content` | `string` | 消息内容 |
| `created_at` | `string` | 消息创建时间 |

### 2.5 TaskSummary

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | `string` | 任务 ID |
| `title` | `string` | 任务标题 |
| `status` | `"pending" \| "in_progress" \| "done" \| "failed"` | 任务状态 |
| `priority` | `"low" \| "medium" \| "high" \| "urgent"` | 优先级 |
| `todo_count` | `number` | Todo 总数 |
| `completed_todo_count` | `number` | 已完成 Todo 数量 |
| `created_at` | `string` | 创建时间 |
| `updated_at` | `string` | 更新时间 |

### 2.6 ConversationListItem

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | `string` | 对话 ID |
| `project_id` | `string` | 所属项目 ID |
| `status` | `"active" \| "resolved"` | 对话状态 |
| `last_message` | [`ConversationMessage`](#24-conversationmessage) | 最后一条消息 |
| `linked_task` | [`TaskSummary`](#25-tasksummary) \| `null` | 若已生成任务，则返回任务摘要 |
| `created_at` | `string` | 创建时间 |
| `updated_at` | `string` | 更新时间 |

### 2.7 ConversationDetail

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | `string` | 对话 ID |
| `project_id` | `string` | 所属项目 ID |
| `status` | `"active" \| "resolved"` | 对话状态 |
| `messages` | [`ConversationMessage[]`](#24-conversationmessage) | 全量消息列表，按时间升序 |
| `linked_task` | [`TaskSummary`](#25-tasksummary) \| `null` | 关联任务摘要 |
| `created_at` | `string` | 创建时间 |
| `updated_at` | `string` | 更新时间 |

### 2.8 Agent

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | `string` | Agent ID |
| `name` | `string` | 名称 |
| `description` | `string` | 描述 |
| `role` | `"pm" \| "developer" \| "reviewer" \| "custom"` | Agent 角色 |
| `capabilities` | `string[]` | 能力标签列表 |
| `node_id` | `string` | ClawSynapse 节点唯一标识 |
| `status` | `"online" \| "offline" \| "busy"` | 在线状态（由 ClawSynapse 节点发现机制维护） |
| `last_seen_at` | `string \| null` | 最近在线时间 |
| `created_at` | `string` | 创建时间 |
| `updated_at` | `string` | 更新时间 |

#### AgentInsights

| 字段 | 类型 | 说明 |
|------|------|------|
| `role` | `"pm" \| "developer" \| "reviewer" \| "custom"` | Agent 角色 |
| `total_items` | `number` | 纳入分析的总工作项数；PM 为任务数，执行型 Agent 为 Todo 数 |
| `active_items` | `number` | 当前 `in_progress` 的工作项数 |
| `pending_over_24h` | `number` | 年龄超过 24 小时且尚未闭环的工作项数；当前口径包含 `pending` 与 `in_progress` |
| `failures_last_7d` | `number` | 最近 7 天失败工作项数 |
| `completions_last_7d` | `number` | 最近 7 天完成工作项数 |
| `oldest_pending_ms` | `number \| null` | 最老 `pending` 工作项年龄，单位毫秒 |
| `longest_in_progress_ms` | `number \| null` | 最长 `in_progress` 工作项执行时长，单位毫秒 |
| `response_p50_ms` | `number \| null` | 响应时长 P50；仅执行型 Agent 有值，口径为 `started_at - created_at` |
| `response_p90_ms` | `number \| null` | 响应时长 P90；仅执行型 Agent 有值 |
| `completion_p50_ms` | `number \| null` | 完成时长 P50；仅执行型 Agent 有值，口径为 `completed_at - started_at` |
| `completion_p90_ms` | `number \| null` | 完成时长 P90；仅执行型 Agent 有值 |
| `aging` | [`AgentAgingBucket[]`](#agentagingbucket) | 老化分布 |
| `priority_breakdown` | [`AgentPriorityBreakdown[]`](#agentprioritybreakdown) | 按优先级聚合的完成情况 |
| `project_contribution` | [`AgentProjectContribution[]`](#agentprojectcontribution) | 项目贡献排行，最多返回前 5 个 |
| `risk_items` | [`AgentRiskItem[]`](#agentriskitem) | 风险项列表，按年龄倒序，最多返回前 4 个 |

#### AgentAgingBucket

| 字段 | 类型 | 说明 |
|------|------|------|
| `label` | `string` | 分桶标签，当前固定为 `1 小时内`、`1-24 小时`、`1-3 天`、`3 天以上` |
| `count` | `number` | 该分桶中的工作项数 |

#### AgentPriorityBreakdown

| 字段 | 类型 | 说明 |
|------|------|------|
| `priority` | `"low" \| "medium" \| "high" \| "urgent"` | 优先级 |
| `label` | `string` | 优先级展示文案 |
| `total` | `number` | 总工作项数 |
| `done` | `number` | 已完成数量 |
| `failed` | `number` | 已失败数量 |
| `pending` | `number` | 待处理数量 |
| `in_progress` | `number` | 进行中数量 |
| `completion_rate` | `number` | 完成率，口径为 `done / (done + failed) * 100`；若无已结束工作项则为 `0` |

#### AgentProjectContribution

| 字段 | 类型 | 说明 |
|------|------|------|
| `project_id` | `string` | 项目 ID |
| `project_name` | `string` | 项目名称 |
| `total` | `number` | 该项目下的总工作项数 |
| `done` | `number` | 已完成数量 |
| `failed` | `number` | 已失败数量 |
| `pending` | `number` | 待处理数量 |
| `in_progress` | `number` | 进行中数量 |
| `completion_rate` | `number` | 项目完成率，口径同上 |

#### AgentRiskItem

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | `string` | 工作项 ID；PM 为 task ID，执行型 Agent 为 todo ID |
| `kind` | `"task" \| "todo"` | 工作项类型 |
| `title` | `string` | 工作项标题 |
| `subtitle` | `string` | 辅助标题；PM 当前固定为 `PM 任务`，执行型 Agent 为所属任务标题 |
| `project_id` | `string` | 项目 ID |
| `project_name` | `string` | 项目名称 |
| `status` | `"pending" \| "in_progress"` | 风险项状态 |
| `age_ms` | `number` | 当前年龄/耗时，单位毫秒 |

说明：

- PM Agent 当前没有独立的任务开始时间，因此 `in_progress` 任务的年龄使用 `task.created_at` 作为起点。
- 执行型 Agent 的响应/完成时长分位数仅统计存在完整时间戳的数据。

### 2.9 TodoAssignee

| 字段 | 类型 | 说明 |
|------|------|------|
| `agent_id` | `string` | 执行 Agent ID |
| `name` | `string` | 执行 Agent 名称 |
| `node_id` | `string` | 执行 Agent 节点 ID |

### 2.10 TodoResultArtifactRef

| 字段 | 类型 | 说明 |
|------|------|------|
| `artifact_id` | `string` | 交付物 ID |
| `kind` | `"file" \| "link" \| "log" \| "report"` | 交付物类型 |
| `label` | `string` | 展示名称 |

### 2.11 Todo

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | `string` | Todo ID，uuid |
| `title` | `string` | 标题 |
| `description` | `string` | 描述 |
| `status` | `"pending" \| "in_progress" \| "done" \| "failed"` | Todo 状态 |
| `assignee` | [`TodoAssignee`](#29-todoassignee) | 执行 Agent 摘要 |
| `started_at` | `string \| null` | 开始时间 |
| `completed_at` | `string \| null` | 完成时间 |
| `failed_at` | `string \| null` | 失败时间 |
| `error` | `string \| null` | 失败原因 |
| `result.summary` | `string` | Todo 执行摘要 |
| `result.output` | `string` | Todo 输出文本 |
| `result.artifact_refs` | [`TodoResultArtifactRef[]`](#210-todoresultartifactref) | Todo 关联的交付物引用 |
| `result.metadata` | `object` | 结构化扩展信息；文件上传时可在 `metadata.transfers` 中附带 `transfer_id`、`size`、`checksum` 等信息 |
| `created_at` | `string` | 创建时间 |

### 2.12 TaskArtifact

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | `string` | 交付物 ID |
| `source_todo_id` | `string \| null` | 来源 Todo ID |
| `kind` | `"file" \| "link" \| "log" \| "report"` | 交付物类型 |
| `title` | `string` | 标题 |
| `uri` | `string` | 文件路径、URL、对象地址，或 `transfer://<transferId>` 形式的传输引用 |
| `mime_type` | `string \| null` | MIME 类型 |
| `metadata` | `object` | 扩展元信息；transfer 型文件会包含 `transfer_id` 与 `transfer` 元数据 |

### 2.13 TaskResult

| 字段 | 类型 | 说明 |
|------|------|------|
| `summary` | `string` | 任务最终摘要 |
| `final_output` | `string` | 最终产出文本 |
| `metadata` | `object` | 扩展元信息 |

### 2.14 TaskListItem

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | `string` | 任务 ID |
| `project_id` | `string` | 所属项目 ID |
| `conversation_id` | `string` | 来源对话 ID |
| `title` | `string` | 标题 |
| `description` | `string` | Markdown 描述 |
| `status` | `"pending" \| "in_progress" \| "done" \| "failed"` | 任务状态 |
| `priority` | `"low" \| "medium" \| "high" \| "urgent"` | 优先级 |
| `pm_agent` | [`PMAgentSummary`](#22-pmagentsummary) | 创建该任务的 PM Agent |
| `todo_count` | `number` | Todo 总数 |
| `completed_todo_count` | `number` | 已完成 Todo 数量 |
| `failed_todo_count` | `number` | 已失败 Todo 数量 |
| `created_at` | `string` | 创建时间 |
| `updated_at` | `string` | 更新时间 |

### 2.15 TaskDetail

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | `string` | 任务 ID |
| `project_id` | `string` | 所属项目 ID |
| `conversation_id` | `string` | 来源对话 ID |
| `title` | `string` | 标题 |
| `description` | `string` | Markdown 描述 |
| `status` | `"pending" \| "in_progress" \| "done" \| "failed"` | 任务状态 |
| `priority` | `"low" \| "medium" \| "high" \| "urgent"` | 优先级 |
| `pm_agent` | [`PMAgentSummary`](#22-pmagentsummary) | 创建任务的 PM Agent |
| `todos` | [`Todo[]`](#211-todo) | Todo 列表 |
| `artifacts` | [`TaskArtifact[]`](#212-taskartifact) | 任务交付物列表 |
| `result` | [`TaskResult`](#213-taskresult) | 任务最终结果 |
| `version` | `number` | 版本号 |
| `created_at` | `string` | 创建时间 |
| `updated_at` | `string` | 更新时间 |

### 2.16 TaskEvent

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | `string` | 事件 ID |
| `task_id` | `string` | 任务 ID |
| `actor_type` | `"user" \| "agent" \| "system"` | 行为发起方类型 |
| `actor_id` | `string` | 行为发起方 ID |
| `event_type` | `"task_created" \| "task_status_changed" \| "todo_assigned" \| "todo_started" \| "todo_progress" \| "todo_completed" \| "todo_failed" \| "task_comment"` | 事件类型 |
| `content` | `string \| null` | 展示文案 |
| `metadata` | `object` | 结构化附加字段 |
| `created_at` | `string` | 创建时间 |

### 2.17 TypeScript 领域类型名建议

建议前端领域模型直接使用以下类型名：

```ts
export type User = { /* 对应 2.1 */ }
export type PMAgentSummary = { /* 对应 2.2 */ }
export type Project = { /* 对应 2.3 */ }
export type ConversationMessage = { /* 对应 2.4 */ }
export type TaskSummary = { /* 对应 2.5 */ }
export type ConversationListItem = { /* 对应 2.6 */ }
export type ConversationDetail = { /* 对应 2.7 */ }
export type Agent = { /* 对应 2.8 */ }
export type TodoAssignee = { /* 对应 2.9 */ }
export type TodoResultArtifactRef = { /* 对应 2.10 */ }
export type Todo = { /* 对应 2.11 */ }
export type TaskArtifact = { /* 对应 2.12 */ }
export type TaskResult = { /* 对应 2.13 */ }
export type TaskListItem = { /* 对应 2.14 */ }
export type TaskDetail = { /* 对应 2.15 */ }
export type TaskEvent = { /* 对应 2.16 */ }
```

## 三、认证接口

### 3.1 注册

`POST /api/v1/auth/register`

请求体：

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `email` | `string` | 是 | 登录邮箱，唯一 |
| `name` | `string` | 是 | 用户名称 |
| `password` | `string` | 是 | 明文密码，由后端加密存储，至少 8 位 |

请求示例：

```json
{
  "email": "lalo@example.com",
  "name": "Lalo",
  "password": "StrongPass123!"
}
```

响应示例：

```json
{
  "data": {
    "token": "jwt-token",
    "user": {
      "id": "65f1234567890abcde000001",
      "email": "lalo@example.com",
      "name": "Lalo",
      "created_at": "2026-03-16T10:30:00Z",
      "updated_at": "2026-03-16T10:30:00Z"
    }
  }
}
```

### 3.2 登录

`POST /api/v1/auth/login`

请求体：

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `email` | `string` | 是 | 登录邮箱 |
| `password` | `string` | 是 | 明文密码 |

请求示例：

```json
{
  "email": "lalo@example.com",
  "password": "StrongPass123!"
}
```

响应示例：

```json
{
  "data": {
    "token": "jwt-token",
    "user": {
      "id": "65f1234567890abcde000001",
      "email": "lalo@example.com",
      "name": "Lalo",
      "created_at": "2026-03-16T10:30:00Z",
      "updated_at": "2026-03-16T10:30:00Z"
    }
  }
}
```

## 四、项目接口

### 4.1 创建项目

`POST /api/v1/projects`

请求体：

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `name` | `string` | 是 | 项目名称 |
| `description` | `string` | 是 | 项目描述 |
| `pm_agent_id` | `string` | 是 | 项目绑定的 PM Agent ID |

响应：

- `201 Created`
- `data` 为 [`Project`](#23-project)

请求示例：

```json
{
  "name": "TrustMesh MVP",
  "description": "多 Agent 协作开发项目",
  "pm_agent_id": "65f1234567890abcde001234"
}
```

### 4.2 项目列表

`GET /api/v1/projects`

查询参数：无

响应：

- `200 OK`
- `data.items` 为 [`Project[]`](#23-project)
- `meta.count` 为项目数量

响应示例：

```json
{
  "data": {
    "items": [
      {
        "id": "65f1234567890abcde010001",
        "name": "TrustMesh MVP",
        "description": "多 Agent 协作开发项目",
        "status": "active",
        "pm_agent": {
          "id": "65f1234567890abcde001234",
          "name": "PM Agent Alpha",
          "node_id": "node-pm-001",
          "status": "online"
        },
        "created_at": "2026-03-16T10:30:00Z",
        "updated_at": "2026-03-16T10:30:00Z"
      }
    ]
  },
  "meta": {
    "count": 1
  }
}
```

### 4.3 项目详情

`GET /api/v1/projects/:id`

路径参数：

| 参数 | 类型 | 说明 |
|------|------|------|
| `id` | `string` | 项目 ID |

响应：

- `200 OK`
- `data` 为 [`Project`](#23-project)

### 4.4 更新项目

`PATCH /api/v1/projects/:id`

路径参数：

| 参数 | 类型 | 说明 |
|------|------|------|
| `id` | `string` | 项目 ID |

请求体：

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `name` | `string` | 否 | 新项目名称 |
| `description` | `string` | 否 | 新项目描述 |

说明：

- 不支持通过该接口修改 `pm_agent_id`

响应：

- `200 OK`
- `data` 为更新后的 [`Project`](#23-project)

### 4.5 归档项目

`DELETE /api/v1/projects/:id`

路径参数：

| 参数 | 类型 | 说明 |
|------|------|------|
| `id` | `string` | 项目 ID |

响应：

- `200 OK`
- `data` 为归档后的 [`Project`](#23-project)
- 返回对象中 `status="archived"`

## 五、对话接口

### 5.1 创建对话并发送首条需求

`POST /api/v1/projects/:projectId/conversations`

路径参数：

| 参数 | 类型 | 说明 |
|------|------|------|
| `projectId` | `string` | 项目 ID |

请求体：

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `content` | `string` | 是 | 首条需求消息 |

响应：

- `201 Created`
- `data` 为 [`ConversationDetail`](#27-conversationdetail)

响应示例：

```json
{
  "data": {
    "id": "65f1234567890abcde020001",
    "project_id": "65f1234567890abcde010001",
    "status": "active",
    "messages": [
      {
        "id": "4f4872df-8c5b-44d1-a5c4-3773b143b4a0",
        "role": "user",
        "content": "我需要一个用户登录功能，支持邮箱密码和 Google OAuth。",
        "created_at": "2026-03-16T10:35:00Z"
      }
    ],
    "linked_task": null,
    "created_at": "2026-03-16T10:35:00Z",
    "updated_at": "2026-03-16T10:35:00Z"
  }
}
```

### 5.2 对话列表

`GET /api/v1/projects/:projectId/conversations`

路径参数：

| 参数 | 类型 | 说明 |
|------|------|------|
| `projectId` | `string` | 项目 ID |

响应：

- `200 OK`
- `data.items` 为 [`ConversationListItem[]`](#26-conversationlistitem)
- 仅返回当前登录用户在该项目下的对话

### 5.3 对话详情

`GET /api/v1/conversations/:id`

路径参数：

| 参数 | 类型 | 说明 |
|------|------|------|
| `id` | `string` | 对话 ID |

响应：

- `200 OK`
- `data` 为 [`ConversationDetail`](#27-conversationdetail)

### 5.4 继续发送消息

`POST /api/v1/conversations/:id/messages`

路径参数：

| 参数 | 类型 | 说明 |
|------|------|------|
| `id` | `string` | 对话 ID |

请求体：

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `content` | `string` | 是 | 用户补充消息 |

响应：

- `200 OK`
- `data` 为最新的 [`ConversationDetail`](#27-conversationdetail)

说明：

- 仅允许 `status=active` 的对话调用
- 若对话已 `resolved`，返回 `409 Conflict` + `CONVERSATION_RESOLVED`

## 六、任务接口

### 6.1 任务列表

`GET /api/v1/projects/:projectId/tasks`

路径参数：

| 参数 | 类型 | 说明 |
|------|------|------|
| `projectId` | `string` | 项目 ID |

查询参数：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `status` | `string` | 否 | 按任务状态筛选，支持 `pending`、`in_progress`、`done`、`failed` |

响应：

- `200 OK`
- `data.items` 为 [`TaskListItem[]`](#214-tasklistitem)

响应示例：

```json
{
  "data": {
    "items": [
      {
        "id": "65f1234567890abcde030001",
        "project_id": "65f1234567890abcde010001",
        "conversation_id": "65f1234567890abcde020001",
        "title": "实现用户登录",
        "description": "支持邮箱密码和 Google OAuth",
        "status": "in_progress",
        "priority": "high",
        "pm_agent": {
          "id": "65f1234567890abcde001234",
          "name": "PM Agent Alpha",
          "node_id": "node-pm-001",
          "status": "online"
        },
        "todo_count": 3,
        "completed_todo_count": 1,
        "failed_todo_count": 0,
        "created_at": "2026-03-16T10:40:00Z",
        "updated_at": "2026-03-16T10:45:00Z"
      }
    ]
  },
  "meta": {
    "count": 1
  }
}
```

### 6.2 任务详情

`GET /api/v1/tasks/:id`

路径参数：

| 参数 | 类型 | 说明 |
|------|------|------|
| `id` | `string` | 任务 ID |

响应：

- `200 OK`
- `data` 为 [`TaskDetail`](#215-taskdetail)

响应示例：

```json
{
  "data": {
    "id": "65f1234567890abcde030001",
    "project_id": "65f1234567890abcde010001",
    "conversation_id": "65f1234567890abcde020001",
    "title": "实现用户登录",
    "description": "支持邮箱密码和 Google OAuth",
    "status": "in_progress",
    "priority": "high",
    "pm_agent": {
      "id": "65f1234567890abcde001234",
      "name": "PM Agent Alpha",
      "node_id": "node-pm-001",
      "status": "online"
    },
    "todos": [
      {
        "id": "todo_1",
        "title": "实现后端登录接口",
        "description": "完成邮箱密码登录 API",
        "status": "done",
        "assignee": {
          "agent_id": "65f1234567890abcde001235",
          "name": "Backend Agent A",
          "node_id": "node-backend-001"
        },
        "started_at": "2026-03-16T10:42:00Z",
        "completed_at": "2026-03-16T10:50:00Z",
        "failed_at": null,
        "error": null,
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
          "metadata": {}
        },
        "created_at": "2026-03-16T10:40:00Z"
      }
    ],
    "artifacts": [
      {
        "id": "artifact_login_api",
        "source_todo_id": "todo_1",
        "kind": "report",
        "title": "登录接口实现说明",
        "uri": "https://example.com/reports/login-api",
        "mime_type": "text/markdown",
        "metadata": {}
      }
    ],
    "result": {
      "summary": "",
      "final_output": "",
      "metadata": {}
    },
    "version": 1,
    "created_at": "2026-03-16T10:40:00Z",
    "updated_at": "2026-03-16T10:50:00Z"
  }
}
```

### 6.3 任务事件流

`GET /api/v1/tasks/:id/events`

路径参数：

| 参数 | 类型 | 说明 |
|------|------|------|
| `id` | `string` | 任务 ID |

响应：

- `200 OK`
- `data.items` 为 [`TaskEvent[]`](#216-taskevent)

### 6.4 查询任务交付物对应的 Transfer

`GET /api/v1/tasks/:id/artifacts/:artifactId/transfer`

路径参数：

| 参数 | 类型 | 说明 |
|------|------|------|
| `id` | `string` | 任务 ID |
| `artifactId` | `string` | 交付物 ID |

说明：

- 仅当该交付物由文件传输生成时可用
- 服务端会基于交付物中的 `transfer_id` 向 ClawSynapse 查询传输详情

响应：

- `200 OK`
- `data` 为 transfer 详情对象，字段以 ClawSynapse `GET /v1/transfer/{transferId}` 返回为准

## 七、Agent 接口

### 7.1 添加 Agent

`POST /api/v1/agents`

请求体：

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `node_id` | `string` | 是 | 节点唯一标识 |
| `name` | `string` | 是 | Agent 名称 |
| `role` | `"pm" \| "developer" \| "reviewer" \| "custom"` | 是 | Agent 角色 |
| `description` | `string` | 是 | 描述 |
| `capabilities` | `string[]` | 是 | 能力标签 |

响应：

- `201 Created`
- `data` 为 [`Agent`](#28-agent)

### 7.2 Agent 列表

`GET /api/v1/agents`

查询参数：无

响应：

- `200 OK`
- `data.items` 为 [`Agent[]`](#28-agent)

### 7.3 Agent 详情

`GET /api/v1/agents/:id`

路径参数：

| 参数 | 类型 | 说明 |
|------|------|------|
| `id` | `string` | Agent ID |

响应：

- `200 OK`
- `data` 为 [`Agent`](#28-agent)

### 7.4 更新 Agent

`PATCH /api/v1/agents/:id`

路径参数：

| 参数 | 类型 | 说明 |
|------|------|------|
| `id` | `string` | Agent ID |

请求体：

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `name` | `string` | 否 | Agent 名称 |
| `role` | `"pm" \| "developer" \| "reviewer" \| "custom"` | 否 | Agent 角色 |
| `description` | `string` | 否 | 描述 |
| `capabilities` | `string[]` | 否 | 能力标签 |

说明：

- 不支持修改 `node_id`

响应：

- `200 OK`
- `data` 为更新后的 [`Agent`](#28-agent)

### 7.5 删除 Agent

`DELETE /api/v1/agents/:id`

路径参数：

| 参数 | 类型 | 说明 |
|------|------|------|
| `id` | `string` | Agent ID |

响应：

- `204 No Content`

说明：

- 仅当该 Agent 未被 `projects.pm_agent_id`、`tasks.pm_agent_id`、`tasks.todos[].assignee_agent_id` 引用时才允许删除
- 若已被引用，返回 `409 Conflict` + `AGENT_IN_USE`

### 7.6 Agent Insights

`GET /api/v1/agents/:id/insights`

路径参数：

| 参数 | 类型 | 说明 |
|------|------|------|
| `id` | `string` | Agent ID |

响应：

- `200 OK`
- `data` 为 [`AgentInsights`](#agentinsights)

字段口径说明：

- `total_items`
  - PM Agent：统计其创建/负责的任务数
  - 执行型 Agent：统计分配给该 Agent 的 Todo 数
- `pending_over_24h`
  - 当前口径为“年龄超过 24 小时且未闭环”的工作项，包含 `pending` 和 `in_progress`
- `oldest_pending_ms`
  - 仅从 `pending` 工作项中取最大年龄
- `longest_in_progress_ms`
  - 仅从 `in_progress` 工作项中取最大执行时长
- `response_p50_ms` / `response_p90_ms`
  - 仅执行型 Agent 有值
  - 口径：`todo.started_at - todo.created_at`
- `completion_p50_ms` / `completion_p90_ms`
  - 仅执行型 Agent 有值
  - 口径：`todo.completed_at - todo.started_at`
- `risk_items`
  - 仅包含 `pending` 与 `in_progress` 工作项
  - 按年龄倒序返回前 4 项

## 八、前端调用建议

### 8.0 建议的 SDK 方法签名

```ts
register(input: AuthRegisterRequest): Promise<ApiResponse<AuthSuccessData>>
login(input: AuthLoginRequest): Promise<ApiResponse<AuthSuccessData>>

createProject(input: CreateProjectRequest): Promise<ApiResponse<Project>>
listProjects(): Promise<ApiListResponse<Project>>
getProject(id: string): Promise<ApiResponse<Project>>
updateProject(id: string, input: UpdateProjectRequest): Promise<ApiResponse<Project>>
archiveProject(id: string): Promise<ApiResponse<Project>>

createConversation(projectId: string, input: CreateConversationRequest): Promise<ApiResponse<ConversationDetail>>
listProjectConversations(projectId: string): Promise<ApiListResponse<ConversationListItem>>
getConversation(id: string): Promise<ApiResponse<ConversationDetail>>
appendConversationMessage(id: string, input: AppendConversationMessageRequest): Promise<ApiResponse<ConversationDetail>>

listProjectTasks(projectId: string, query?: ListProjectTasksQuery): Promise<ApiListResponse<TaskListItem>>
getTask(id: string): Promise<ApiResponse<TaskDetail>>
listTaskEvents(id: string): Promise<ApiListResponse<TaskEvent>>

createAgent(input: CreateAgentRequest): Promise<ApiResponse<Agent>>
listAgents(): Promise<ApiListResponse<Agent>>
getAgent(id: string): Promise<ApiResponse<Agent>>
updateAgent(id: string, input: UpdateAgentRequest): Promise<ApiResponse<Agent>>
deleteAgent(id: string): Promise<void>
```

### 8.1 建议的请求类型名

```ts
export interface AuthRegisterRequest {
  email: string
  name: string
  password: string
}

export interface AuthLoginRequest {
  email: string
  password: string
}

export interface AuthSuccessData {
  token: string
  user: User
}

export interface CreateProjectRequest {
  name: string
  description: string
  pm_agent_id: string
}

export interface UpdateProjectRequest {
  name?: string
  description?: string
}

export interface CreateConversationRequest {
  content: string
}

export interface AppendConversationMessageRequest {
  content: string
}

export interface ListProjectTasksQuery {
  status?: "pending" | "in_progress" | "done" | "failed"
}

export interface CreateAgentRequest {
  node_id: string
  name: string
  role: "pm" | "developer" | "reviewer" | "custom"
  description: string
  capabilities: string[]
}

export interface UpdateAgentRequest {
  name?: string
  role?: "pm" | "developer" | "reviewer" | "custom"
  description?: string
  capabilities?: string[]
}
```

### 8.2 ConversationPage

- 先调 `GET /api/v1/projects/:projectId/conversations`
- 若没有 `active` 对话，首发消息走 `POST /api/v1/projects/:projectId/conversations`
- 若有 `active` 对话，发消息走 `POST /api/v1/conversations/:id/messages`
- 轮询 `GET /api/v1/conversations/:id` 读取 PM 回复和 `linked_task`

### 8.3 ProjectBoardPage

- 调 `GET /api/v1/projects/:projectId/tasks?status=...` 拉取看板列数据
- 点开任务后调 `GET /api/v1/tasks/:id`
- 时间线单独调 `GET /api/v1/tasks/:id/events`

### 8.4 AgentListPage

- 调 `GET /api/v1/agents` 展示名称、角色、能力、状态、最近心跳时间
- 新增、编辑成功后，刷新 `GET /api/v1/agents`
