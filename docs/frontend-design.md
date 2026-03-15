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
| `/projects/:id` | ProjectBoardPage | 看板视图（核心页面） |
| `/projects/:id/tasks/:id` | TaskDrawer | 任务详情（侧边 Sheet） |
| `/projects/:id/chat` | ConversationPage | 与 PM Agent 对话 |
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
│   │   ├── BoardColumn              # 状态列 (Todo/InProgress/Done/Failed)
│   │   │   └── TaskCard             # shadcn Card
│   │   │       ├── PriorityBadge    # shadcn Badge
│   │   │       ├── AssigneeBadge    # Agent 机器人图标 / 用户头像
│   │   │       ├── TodoProgress     # Todo 完成进度条
│   │   │       └── StatusIndicator
│   │   └── TaskSheet                # shadcn Sheet（右侧抽屉）
│   │       ├── TaskHeader           # 标题、状态、优先级
│   │       ├── TaskDescription      # Markdown 编辑/展示
│   │       ├── TaskAssignment       # Agent/用户选择器（shadcn Select）
│   │       ├── TodoList             # Todo 清单（复杂任务显示）
│   │       │   └── TodoItem         # 单个 Todo + 状态 + 指派 + 结果
│   │       ├── TaskTimeline         # 事件活动流
│   │       └── TaskResult           # Agent 结果展示
│   │
│   ├── ConversationPage             # 与 PM Agent 对话
│   │   ├── MessageList              # 消息列表（shadcn ScrollArea）
│   │   │   └── MessageBubble        # 消息气泡（区分用户/PM）
│   │   ├── MessageInput             # 输入框 + 发送按钮
│   │   └── PlanPreview              # PM 生成的任务预览
│   │
│   └── AgentListPage
│       ├── AgentCard                # shadcn Card + 状态指示灯
│       └── AgentConfigDialog        # shadcn Dialog
│
└── Shared (shadcn/ui 基础组件)
    ├── Button / Input / Badge / Avatar
    ├── Card / Dialog / Sheet / Select
    ├── ScrollArea / Tabs / Separator
    └── EmptyState（自定义）
```
