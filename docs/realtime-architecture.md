# 实时推送架构

本文描述 TrustMesh 当前已经落地的前后端实时推送架构，目标是统一消息推送设计，避免页面各自轮询、各自接流。

适用范围：

- 前端收件箱、侧栏未读数
- Dashboard 最近活动 / 最近任务 / 统计
- 项目任务列表
- 会话列表与会话详情
- 任务详情、任务活动、任务评论
- Agent 在线状态

## 一、设计目标

当前实时方案遵循两层分工：

1. 快照查询
   - HTTP Query 负责首屏加载和显式刷新。
2. 实时增量
   - SSE 负责推送权威状态变化。
3. 降级兜底
   - 仅当全局实时流不可用时，详情页启用低频轮询。

核心原则：

- 不允许同一份页面数据长期同时依赖“高频轮询 + 高频实时流”。
- 全局聚合视图和详情页统一消费用户级事件流。
- React Query 是前端唯一缓存真相源，实时层只做 patch 或 invalidate。

## 二、实时分层

### 2.1 用户级全局流

后端入口：

- `GET /api/v1/events/stream`

职责：

- 通知创建与已读同步
- 任务更新、任务事件、任务评论
- 会话更新
- Agent 状态变化
- Dashboard / 列表类聚合视图的缓存同步

前端接入：

- `frontend/src/realtime/provider.tsx`

特点：

- 用户登录后建立单条全局 SSE 连接
- 未登录态不连接
- 页面不直接订阅该流，统一由 provider 分发事件

### 2.2 降级轮询

适用场景：

- 全局 SSE 处于 `reconnecting` 或 `disconnected`
- 当前资源仍然是活跃态，例如 `task.status in [pending, in_progress]` 或 `conversation.status=active`

原则：

- 仅详情 query 可启用降级轮询
- Dashboard、通知、列表页不使用常态轮询

## 三、当前事件类型

当前用户级事件流已实现以下事件：

| 事件类型 | 说明 |
|------|------|
| `notification.created` | 新通知创建 |
| `notification.read` | 单条通知已读 |
| `notifications.all_read` | 批量全部已读 |
| `task.updated` | 任务详情快照变化 |
| `task.event.created` | 任务活动流新增事件 |
| `task.comment.created` | 任务评论新增 |
| `conversation.updated` | 会话详情变化 |
| `agent.status.changed` | Agent 在线状态变化 |

事件信封统一字段：

```json
{
  "id": "evt_xxx",
  "type": "task.updated",
  "occurred_at": "2026-03-21T02:00:00Z",
  "payload": {}
}
```

约束：

- `type` 必须稳定，不允许同义重名。
- `occurred_at` 使用服务端 UTC 时间。
- `payload` 必须是面向前端 reducer 可直接消费的结构。

## 四、前端缓存同步规则

### 4.1 patch 优先

优先 patch 的场景：

- 单条通知状态变化
- 单个任务详情更新
- 单个会话详情更新
- 单个任务评论新增
- 单个任务活动新增
- 单个 Agent 状态变更

原因：

- 数据定位明确
- 可避免无意义重拉
- UI 反馈更快

### 4.2 invalidate 兜底

优先 invalidate 的场景：

- Dashboard 统计卡片
- 受 Agent 状态影响的聚合数据
- 不易稳定局部 patch 的复杂聚合视图

当前实现中：

- 正常收流时，优先 patch 已知缓存
- 全局 SSE 重连成功后，对 `notifications`、`dashboard`、`projects`、`agents`、`tasks`、`conversations` 走定向失效，补齐断线期间可能漏掉的变更

### 4.3 主要同步关系

| 事件类型 | 主要影响缓存 |
|------|------|
| `notification.created` | `notifications/*`、`notifications/unread-count` |
| `notification.read` | `notifications/*`、`notifications/unread-count` |
| `notifications.all_read` | `notifications/*`、`notifications/unread-count` |
| `task.updated` | `tasks/detail`、项目任务列表、`dashboard/tasks`、`dashboard/stats` |
| `task.event.created` | `tasks/events`、`dashboard/events`、`agents/:id/events` |
| `task.comment.created` | `tasks/comments` |
| `conversation.updated` | `conversations/detail`、项目会话列表 |
| `agent.status.changed` | `agents`、`agents/:id`、`dashboard/events`、`dashboard/stats`，必要时失效 `projects` / `tasks` |

补充约束：

- 项目卡片和项目页头部如果展示的是“由任务聚合得出的工作态”，则该状态属于 `task.updated` 的派生缓存。
- 前端应优先根据 `task.updated` patch `projects` / `projects/:id` 中的项目摘要，而不是等待页面重查。
- 当 reducer 无法确认前置任务状态时，可以对受影响的 `projects` 做定向 invalidate，作为 patch 失败兜底。
- `agent.status.changed` 如果只影响项目绑定 PM 的在线状态，优先 patch 项目缓存中的 `pm_agent`，而不是直接全量失效项目列表。

## 五、目录约定

前端实时逻辑集中在：

```txt
frontend/src/realtime/
  types.ts
  provider.tsx
  hooks/
    useRealtimeStatus.ts
  reducers/
    notifications.ts
    tasks.ts
    conversations.ts
    agents.ts
```

规则：

- 页面组件不得直接建立全局 SSE 连接。
- 页面组件不得自行建立详情级 SSE 连接。
- 页面组件尽量不直接写 `queryClient.setQueryData`。
- 新增实时事件时，优先在 `realtime/reducers/` 中处理。

## 六、扩展约束

后续新增实时事件，必须同时满足：

1. 后端定义稳定事件类型与 payload。
2. 前端在 `realtime/types.ts` 补充类型。
3. 前端新增或修改对应 reducer。
4. 明确该事件是 patch 还是 invalidate。
5. 至少补一条后端流测试。

禁止做法：

- 在页面组件里单独加 `setInterval` 轮询替代事件流。
- 在多个页面分别建立同一类全局 SSE 连接。
- 只发事件，不定义缓存同步规则。
- 把“通知事件”和“详情快照”混成同一种 payload 结构。

## 七、已知边界

当前已知边界如下：

- 前端构建仍存在较大的 bundle warning，这与实时架构无直接关系。
- 当前前端以单条用户级全局流作为唯一实时入口。
- 详情页仍保留降级轮询能力，但只在全局流异常时启用。

## 八、后续建议

后续演进建议按以下方向推进：

1. 保持事件类型收敛，避免扩散成页面私有协议。
2. 新增实时能力时，优先扩充用户级事件流，避免再引入页面私有连接。
3. 如果后续聚合视图越来越复杂，再考虑引入更系统的事件版本号或更细粒度 reducer 分层。
4. 若未来要支持多端同步或离线补偿，再评估事件顺序、去重和版本控制策略。

## 九、相关实现

后端：

- `backend/internal/handler/realtime.go`
- `backend/internal/handler/sse.go`
- `backend/internal/store/user_streams.go`
- `backend/internal/store/store_events.go`
- `backend/internal/store/notification.go`
- `backend/internal/store/store_notification_internal.go`
- `backend/internal/store/streams.go`

前端：

- `frontend/src/realtime/provider.tsx`
- `frontend/src/realtime/types.ts`
- `frontend/src/realtime/reducers/*.ts`
- `frontend/src/hooks/useNotifications.ts`
- `frontend/src/hooks/useTasks.ts`
- `frontend/src/hooks/useConversations.ts`
