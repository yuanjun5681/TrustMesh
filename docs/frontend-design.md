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

- 新增 Agent：用户在 `AgentConfigDialog` 中填写 `node_id`、名称、role、描述、capabilities、type、config。
- 编辑 Agent：允许修改名称、role、描述、capabilities、config；`node_id` 在编辑态只读展示。
- Agent 卡片展示字段：`name`、`role`、`node_id`、`capabilities`、`status`、`description`。
- 项目创建或编辑 PM Agent 绑定时，从当前用户已添加的 Agent 列表中选择。

## 异常态

- 如果项目 PM Agent 离线，`ConversationPage` 禁止发送需求，并展示明确提示。
- `ProjectListPage` 和项目头部需要展示 PM Agent 当前在线状态。
- `AgentListPage` 需要展示最近心跳时间，便于用户判断节点是否正常在线。
