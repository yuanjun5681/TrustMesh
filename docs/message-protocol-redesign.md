# 消息协议精简改造提案

本文是对现有 TrustMesh Agent 消息协议的精简改造提案，不代表当前已上线实现。

目标：
- 消除“自己发出的状态，又被服务端回推给自己”的回声通知
- 保留 PM Agent 所需的服务端确认与进度感知
- 在不引入通用 `command.result` 的前提下，先完成一版更清晰的消息矩阵
- 为未来如需引入统一 ack 留出扩展空间

## 一、核心判断

当前协议里有两类语义被混在了一起：

1. 命令提交
   - Agent 向 TrustMesh 提交业务动作，如 `task.create`、`todo.progress`
2. 领域事件
   - TrustMesh 将权威状态变化通知给相关方，如 `task.updated`、`todo.updated`

其中最不合理的点在于：
- assignee 发送 `todo.progress` / `todo.complete` / `todo.fail` 后，TrustMesh 又把 `todo.updated` 回发给同一个 assignee
- 这条回推通常不提供比原始命令更多的信息，容易形成“协议回声”

因此，本提案主张：
- 保留 `task.created -> PM Agent`
- 保留任务和 Todo 的状态事件
- 去掉发回“本次状态变更操作者”的自回声通知
- 暂不引入新的通用 ack 消息类型

## 二、设计原则

1. TrustMesh 仍然是唯一业务真相源，所有状态先落库，再向外发事件。
2. 命令和事件严格区分，不让同一条消息同时承担两种职责。
3. 默认不向“本次动作的发起者”回推同义状态事件。
4. 只有当状态变化对接收方是“新信息”时，才发送通知。
5. 事件 payload 需补充最少但关键的溯源字段，支持去重、审计和乱序处理。

## 三、推荐消息矩阵

| 消息类型 | 方向 | 发送方 | 接收方 | 是否回给原发送者 | 用途 |
|------|------|------|------|------|------|
| `conversation.message` | TrustMesh -> Agent | TrustMesh | PM Agent | 否 | 用户需求进入 PM |
| `conversation.reply` | Agent -> TrustMesh | PM Agent | TrustMesh | 否 | PM 回复用户 |
| `task.create` | Agent -> TrustMesh | PM Agent | TrustMesh | 否 | 提交建任务命令 |
| `task.created` | TrustMesh -> Agent | TrustMesh | PM Agent | 是 | 任务创建成功确认，返回服务端任务标识 |
| `task.status_changed` | TrustMesh -> Agent | TrustMesh | PM Agent | 否 | 任务聚合状态变化 |
| `todo.assigned` | TrustMesh -> Agent | TrustMesh | assignee Agent | 否 | 通知某个 Todo 开始执行 |
| `todo.progress` | Agent -> TrustMesh | assignee Agent | TrustMesh | 否 | 提交执行进度 |
| `todo.complete` | Agent -> TrustMesh | assignee Agent | TrustMesh | 否 | 提交完成结果 |
| `todo.fail` | Agent -> TrustMesh | assignee Agent | TrustMesh | 否 | 提交失败结果 |
| `todo.status_changed` | TrustMesh -> Agent | TrustMesh | PM Agent、受影响相关方 | 默认否 | Todo 权威状态变化 |

说明：
- `task.created` 保留，因为它承担的是“服务端确认任务已创建”的语义，而不是普通状态广播。
- `task.updated` 建议重命名为 `task.status_changed`。
- `todo.updated` 建议重命名为 `todo.status_changed`。
- `todo.status_changed` 默认不发给触发该次状态变化的 assignee。

## 四、通知规则

### 4.1 PM Agent

PM Agent 接收：
- `conversation.message`
- `task.created`
- `task.status_changed`
- `todo.status_changed`

理由：
- PM 负责和用户交互，需要看到任务是否真正创建成功
- PM 需要掌握全局任务进展，以决定是否向用户汇报

### 4.2 执行 Agent

执行 Agent 接收：
- `todo.assigned`
- `todo.status_changed`，但仅限“不是自己刚刚触发”的状态变化

可通知执行 Agent 的情况：
- 用户手动派发 Todo
- 系统重派 Todo
- 管理员或系统取消 Todo
- 服务端将 Todo 改为其他状态

不通知执行 Agent 的情况：
- 该 Agent 自己刚发出 `todo.progress`
- 该 Agent 自己刚发出 `todo.complete`
- 该 Agent 自己刚发出 `todo.fail`

## 五、建议事件字段

在不新增消息类型的前提下，建议给状态事件补充以下字段：

### 5.1 task.status_changed

```json
{
  "task_id": "task_123",
  "status": "in_progress",
  "actor_node_id": "node-dev-001",
  "cause": "todo.progress",
  "version": 3
}
```

字段说明：
- `actor_node_id`：谁触发了这次状态变化
- `cause`：由哪个命令或系统动作导致
- `version`：服务端权威版本号，用于去重与乱序处理

### 5.2 todo.status_changed

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

字段说明：
- `task_id` / `todo_id`：资源标识
- `status`：Todo 权威状态
- `actor_node_id`：本次变化的操作者
- `cause`：变化来源，例如 `todo.progress`、`todo.complete`、`manual_dispatch`
- `version`：服务端版本号
- `message`：可选，面向人类的补充说明

## 六、为什么先不引入 command.result

统一 ack 消息如 `command.result` 的优点是通用、规范、易扩展，但它会带来新的复杂度：
- 新增消息类型
- 新增字段规范，如 `request_id`、`status`、`error`
- Agent 端要实现“命令回执”和“领域事件”两套处理逻辑

在当前阶段，如果系统还没有以下明确需求，则不建议先上：
- 命令级确认与失败重试
- 断线重发与强幂等
- 复杂补偿流
- 高强度消息追踪

因此本提案选择：
- 先把领域事件设计干净
- 去掉回声通知
- 只保留真正有意义的服务端确认：`task.created`

后续若要增强可靠性，再引入统一的 `command.result` 也不晚。

## 七、迁移建议

建议按以下顺序演进：

1. 保留现有 `task.created`。
2. 将 `task.updated` 更名为 `task.status_changed`。
3. 将 `todo.updated` 更名为 `todo.status_changed`。
4. 补充 `actor_node_id`、`cause`、`version` 字段。
5. 停止向本次操作者回推 `todo.status_changed`。
6. 仅在“别人或系统修改了该 Todo”时通知 assignee。

## 八、与当前实现的关系

当前实现中：
- `task.created -> PM Agent` 是合理的，应保留。
- `task.updated -> PM Agent` 的职责合理，但建议重命名为 `task.status_changed`。
- `todo.updated -> assignee + PM Agent` 偏冗余，建议改为定向通知。

因此，本提案不是推翻现有架构，而是做一轮协议语义收敛：
- 保留已有的正确部分
- 收紧有噪音的通知
- 让消息名和接收规则更符合真实职责
