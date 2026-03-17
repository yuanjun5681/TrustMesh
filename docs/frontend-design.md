# 前端设计

## UI 技术方案

- **shadcn/ui**：基于 Radix UI 的高质量组件库，可定制主题
- **Tailwind CSS v4**：样式引擎
- 核心 shadcn 组件：Button、Input、Dialog、Sheet、Card、Badge、Avatar、Tabs、ScrollArea、DropdownMenu、Select、Textarea、Separator

## 页面（7 个）

| 路由 | 页面 | 说明 |
|------|------|------|
| `/login` | LoginPage | 登录 |
| `/register` | RegisterPage | 注册 |
| `/projects` | ProjectListPage | 项目列表 |
| `/projects/:projectId` | ProjectBoardPage | 看板视图（核心页面） |
| `/projects/:projectId/tasks/:taskId` | TaskDrawer | 任务详情（侧边 Sheet） |
| `/projects/:projectId/chat` | ConversationPage | 与 PM Agent 对话 |
| `/agents` | AgentListPage | Agent 管理 |

## 组件树

```
App
├── Layout
│   ├── Sidebar                      # 左侧导航（shadcn ScrollArea）
│   │   ├── ProjectNav               # 项目列表导航
│   │   └── AgentNav                  # Agent 管理入口
│   └── MainContent
│
├── Pages
│   ├── LoginPage                    # shadcn Card + Input + Button
│   ├── RegisterPage
│   │
│   ├── ProjectListPage
│   │   └── ProjectCard              # shadcn Card
│   │       └── PMAgentBadge         # 显示项目经理 Agent
│   │
│   ├── ProjectBoardPage             # 核心页面
│   │   ├── ProjectHeader            # 项目名称 + 进入对话按钮
│   │   ├── BoardColumn              # 状态列 (Pending/InProgress/Done/Failed)
│   │   │   └── TaskCard             # shadcn Card
│   │   │       ├── PriorityBadge    # shadcn Badge
│   │   │       ├── AssigneeBadge    # Agent 机器人图标 / 用户头像
│   │   │       ├── TodoProgress     # Todo 完成进度条
│   │   │       └── StatusIndicator
│   │   └── TaskSheet                # shadcn Sheet（右侧抽屉）
│   │       ├── TaskHeader           # 标题、状态、优先级
│   │       ├── TaskDescription      # Markdown 展示
│   │       ├── TodoList             # Todo 清单
│   │       │   └── TodoItem         # 单个 Todo + 状态 + 执行 Agent + 结果
│   │       ├── TaskTimeline         # 事件活动流
│   │       └── TaskResult           # Agent 结果展示
│   │
│   ├── ConversationPage             # 与 PM Agent 对话
│   │   ├── MessageList              # 消息列表（shadcn ScrollArea）
│   │   │   └── MessageBubble        # 消息气泡（区分用户/PM）
│   │   ├── MessageInput             # 输入框 + 发送按钮
│   │   └── PlanPreview              # PM 生成的任务预览（只读）
│   │
│   └── AgentListPage
│       ├── AgentToolbar             # 添加 Agent 按钮 + role/capability 筛选
│       ├── AgentCard                # shadcn Card + 状态指示灯 + node_id
│       └── AgentConfigDialog        # shadcn Dialog（按 node_id 添加/编辑 Agent）
│
└── Shared (shadcn/ui 基础组件)
    ├── Button / Input / Badge / Avatar
    ├── Card / Dialog / Sheet / Select
    ├── ScrollArea / Tabs / Separator
    └── EmptyState（自定义）
```

## Agent 管理交互

- 新增 Agent：用户在 `AgentConfigDialog` 中填写 `node_id`、名称、role、描述、capabilities。
- 编辑 Agent：允许修改名称、role、描述、capabilities；`node_id` 在编辑态只读展示。
- Agent 卡片展示字段：`name`、`role`、`node_id`（ClawSynapse nodeId）、`capabilities`、`status`、`description`。
- 项目创建时，从当前用户已添加的 Agent 列表中选择 PM Agent；MVP 不支持在项目编辑页更换 PM Agent。

## ConversationPage 数据流

- 页面进入时先调用 `GET /api/v1/projects/:projectId/conversations`，读取当前用户在该项目下的对话列表，并优先选中最近一个 `status=active` 的对话。
- 如果不存在活跃对话，输入框第一次提交时调用 `POST /api/v1/projects/:projectId/conversations`，由后端同时创建对话并写入首条用户需求消息。
- 如果已存在活跃对话，后续发送消息调用 `POST /api/v1/conversations/:id/messages`。
- 选中某个对话后，页面调用 `GET /api/v1/conversations/:id` 获取完整消息历史、对话状态和关联任务摘要。
- PM Agent 的回复不会经由前端发送；前端只负责读取由后端回写到 `conversations.messages` 的 `pm_agent` 消息。

## ConversationPage 轮询策略

- MVP 不使用 WebSocket，`ConversationPage` 需要轮询 `GET /api/v1/conversations/:id` 获取 PM 回复和状态变化。
- 当对话为 `active` 且页面处于前台时，建议每 2-3 秒轮询一次当前对话详情。
- 页面失焦、切后台或网络较差时，建议降频到 10-15 秒；离开聊天页后停止轮询。
- 用户发送消息成功后，应立即触发一次详情刷新，而不是等待下一次轮询周期。
- 当对话状态变为 `resolved` 时，停止高频轮询消息，保留对关联任务摘要的展示，并引导用户跳转到任务详情或看板页查看执行进度。

## ConversationPage 交互约定

- `MessageList` 按 `messages.created_at` 升序展示，区分 `user` 和 `pm_agent` 两种气泡样式。
- 发送中消息可以先本地乐观插入，待接口成功后用服务端返回的消息列表或详情结果校正。
- `PlanPreview` 仅在对话已关联 Task 时展示，数据来自 `GET /api/v1/conversations/:id` 返回的任务摘要。
- 若 `GET /api/v1/conversations/:id` 显示对话已 `resolved`，输入框进入只读态，并提示“该需求已沉淀为任务”。
- 如果项目 PM Agent 离线，聊天页不允许新建对话或继续发送消息，并直接展示后端返回的业务错误。

## 异常态

- 如果项目 PM Agent 离线（不在 ClawSynapse peers 列表中），`ConversationPage` 禁止发送需求，并展示明确提示。
- `ProjectListPage` 和项目头部需要展示 PM Agent 当前在线状态。
- `AgentListPage` 需要展示 Agent 最近在线时间（`last_seen_at`），便于用户判断节点是否正常在线。Agent 在线状态由 ClawSynapse 节点发现机制维护。
