# 项目结构与实施计划

## 目录结构

```
TrustMesh/
├── README.md
├── docker-compose.yml              # MongoDB + NATS
├── Makefile
│
├── backend/
│   ├── cmd/
│   │   └── server/
│   │       └── main.go             # 入口，初始化 Gin + MongoDB + NATS
│   ├── internal/
│   │   ├── config/
│   │   │   └── config.go           # 配置加载（env vars）
│   │   ├── model/                  # MongoDB 文档模型
│   │   │   ├── user.go
│   │   │   ├── agent.go
│   │   │   ├── project.go
│   │   │   ├── task.go             # 含 Todo 嵌套结构
│   │   │   ├── task_event.go
│   │   │   ├── agent_session.go
│   │   │   └── conversation.go
│   │   ├── repository/             # 数据访问层
│   │   │   ├── user.go
│   │   │   ├── agent.go
│   │   │   ├── project.go
│   │   │   ├── task.go
│   │   │   ├── task_event.go
│   │   │   └── conversation.go
│   │   ├── handler/                # Gin Handler
│   │   │   ├── auth.go
│   │   │   ├── project.go
│   │   │   ├── task.go
│   │   │   ├── task_todo.go        # Todo API
│   │   │   ├── agent.go
│   │   │   └── conversation.go     # 对话 API
│   │   ├── middleware/
│   │   │   ├── auth.go             # JWT 认证（前端用）
│   │   │   ├── cors.go
│   │   │   └── logging.go
│   │   ├── service/
│   │   │   ├── auth.go
│   │   │   ├── project.go
│   │   │   ├── task.go             # 状态机 + 乐观锁
│   │   │   ├── agent.go
│   │   │   └── conversation.go
│   │   ├── nats/                   # NATS 集成
│   │   │   ├── handler.go          # 订阅处理（Agent → 后端）
│   │   │   ├── rpc.go              # Request-Reply 处理（Agent 查询）
│   │   │   └── publisher.go        # 通知发布（后端 → Agent）
│   │   └── dto/                    # 请求/响应 DTO
│   │       ├── auth.go
│   │       ├── project.go
│   │       ├── task.go
│   │       ├── agent.go
│   │       └── conversation.go
│   ├── go.mod
│   └── go.sum
│
├── frontend/
│   ├── index.html
│   ├── package.json
│   ├── tsconfig.json
│   ├── vite.config.ts
│   ├── components.json             # shadcn/ui 配置
│   ├── src/
│   │   ├── main.tsx
│   │   ├── App.tsx
│   │   ├── app.css                 # Tailwind v4 入口
│   │   ├── lib/
│   │   │   └── utils.ts            # shadcn cn() 工具
│   │   ├── components/
│   │   │   └── ui/                 # shadcn/ui 组件（自动生成）
│   │   │       ├── button.tsx
│   │   │       ├── input.tsx
│   │   │       ├── card.tsx
│   │   │       ├── dialog.tsx
│   │   │       ├── sheet.tsx
│   │   │       ├── badge.tsx
│   │   │       ├── avatar.tsx
│   │   │       ├── select.tsx
│   │   │       ├── scroll-area.tsx
│   │   │       ├── tabs.tsx
│   │   │       ├── separator.tsx
│   │   │       ├── dropdown-menu.tsx
│   │   │       └── textarea.tsx
│   │   ├── api/
│   │   │   ├── client.ts           # ky 实例配置
│   │   │   ├── auth.ts
│   │   │   ├── projects.ts
│   │   │   ├── tasks.ts
│   │   │   ├── agents.ts
│   │   │   └── conversations.ts
│   │   ├── hooks/
│   │   │   ├── useProjects.ts
│   │   │   ├── useTasks.ts
│   │   │   ├── useAgents.ts
│   │   │   └── useConversations.ts
│   │   ├── stores/
│   │   │   └── authStore.ts
│   │   ├── pages/
│   │   │   ├── LoginPage.tsx
│   │   │   ├── RegisterPage.tsx
│   │   │   ├── ProjectListPage.tsx
│   │   │   ├── ProjectBoardPage.tsx
│   │   │   ├── ConversationPage.tsx
│   │   │   └── AgentListPage.tsx
│   │   ├── components/
│   │   │   ├── layout/
│   │   │   │   ├── Sidebar.tsx
│   │   │   │   └── MainLayout.tsx
│   │   │   ├── board/
│   │   │   │   ├── BoardColumn.tsx
│   │   │   │   └── TaskCard.tsx
│   │   │   ├── task/
│   │   │   │   ├── TaskSheet.tsx
│   │   │   │   ├── TaskTimeline.tsx
│   │   │   │   ├── TaskResult.tsx
│   │   │   │   └── TodoList.tsx
│   │   │   ├── conversation/
│   │   │   │   ├── MessageList.tsx
│   │   │   │   ├── MessageBubble.tsx
│   │   │   │   ├── MessageInput.tsx
│   │   │   │   └── PlanPreview.tsx
│   │   │   └── agent/
│   │   │       ├── AgentCard.tsx
│   │   │       └── AgentConfigDialog.tsx
│   │   └── types/
│   │       └── index.ts
│   └── public/
│
└── docs/
    ├── mvp-design.md               # 总览
    ├── data-model.md               # 数据模型
    ├── api-design.md               # API 设计
    ├── agent-engine.md             # Agent 引擎
    ├── frontend-design.md          # 前端设计
    └── project-structure.md        # 本文件
```

## 实施计划

### 阶段 1：基础骨架

- 初始化 Go module（Gin）+ Vite 项目（shadcn/ui）
- docker-compose 配置 MongoDB + NATS
- MongoDB 连接 + 集合初始化 + 索引创建
- NATS 连接 + Publisher 封装
- 认证中间件（JWT + Agent Key）
- 前端路由 + MainLayout + Sidebar

### 阶段 2：项目与任务 CRUD

- 后端 Project + Task CRUD API（含 Todo）
- 前端项目列表 + 看板视图
- TaskSheet 任务详情（含 TodoList）

### 阶段 3：PM Agent 与对话

- Conversation API（对话 CRUD + 消息收发）
- PM Agent 专用 API（创建任务、管理 Todo、指派）
- NATS 通知集成（对话消息 → PM Agent）
- 前端 ConversationPage（对话界面 + 任务预览）

### 阶段 4：Agent 执行体系

- Agent 注册管理 API
- Agent 执行 API（claim/progress/complete/fail + Todo 更新）
- NATS 通知集成（任务指派 → Agent，任务完成 → PM Agent）
- 前端 Agent 管理页 + 任务指派交互

### 阶段 5：活动流与打磨

- TaskEvent 写入和查询
- TaskTimeline 组件
- Agent 结果展示 + Todo 进度可视化
- UI 打磨和错误处理

## 关键设计决策

### 选择 MongoDB

- **Schema 灵活**：Task 内嵌 Todos、Agent config、执行 result 等结构天然适合文档模型
- **嵌套文档**：Todos 直接内嵌在 Task 中，无需 JOIN
- **多态存储**：不同类型 Agent 的 config 结构差异大，文档模型更自然
- **开发简单**：无需迁移文件、无需 ORM 映射、Go struct 直接序列化

### 选择 Gin

- 成熟稳定，社区活跃
- 中间件生态丰富（CORS、日志、Recovery）
- 路由分组便于按模块组织（v1/auth、v1/projects、v1/agent）

### Agent 全走 NATS

- Agent 与后端的所有通信统一通过 NATS，不调用 REST API
- **Publish**：Agent 发布动作（领取、提交、回复等），后端订阅处理
- **Subscribe**：Agent 接收通知（新任务、状态变更等），后端发布
- **Request-Reply**：Agent 查询数据（任务详情、待办列表等），后端响应
- REST API 只服务前端 UI
- Agent 侧只需一个 NATS 连接，实现更简单、更解耦
- 一切通信经过后端中转，保证数据一致性和可追溯性

### PM Agent 角色

- 每个项目绑定一个 PM Agent，负责需求沟通 → 任务规划 → 指派分发 → 结果汇总
- 用户不直接创建任务，而是通过与 PM Agent 对话来驱动
- PM Agent 有独立的 API 端点和更高权限

### 简单任务 vs 复杂任务

- **简单任务**：`todos` 为空数组，直接指派给一个 Agent 完成
- **复杂任务**：`todos` 包含待办清单，Agent 按 Todo 逐项执行
- Todo 嵌入 Task 文档，原子性更新

### 多态指派

使用 `assignee_type` + `assignee_id` 组合。概念统一，查询简单。在 service 层做验证。

### Agent 是外部进程

后端不内置 Agent 运行时。Agent 可以是任何语言实现、运行在任何地方。平台只负责任务编排，不负责 Agent 执行。
