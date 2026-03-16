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
    ├── message-protocol.md         # NATS 消息传输规范
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
- 认证中间件（JWT + NATS Agent 凭证）
- Agent 管理基础模型与 `node_id` 唯一约束
- Agent 心跳上报与在线状态判定
- 前端路由 + MainLayout + Sidebar

### 阶段 2：需求进入 PM

- 后端 Project + Conversation API
- 用户发送需求消息前，校验项目 PM Agent 在线
- 用户发送需求消息后，通过 NATS 通知项目 PM Agent
- 前端项目列表 + 对话入口 + 基础看板

### 阶段 3：PM 创建任务

- PM Agent 通过 NATS 发布 `task.create`
- 后端订阅 `task.create`，写入 Task + Todos
- 服务端按 Todo 的 assignee 发布 `todo.assigned`

### 阶段 4：Agent 执行体系

- Agent 注册管理 API（按 `node_id` 添加、编辑名称/role/描述/能力）
- Agent 执行协议（todo.progress / todo.complete / todo.fail）
- NATS 通知集成（Todo 指派 → Agent，执行结果 → 后端）
- 前端 Agent 管理页 + 任务详情结果展示

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
- **Publish**：PM 发布任务，执行 Agent 发布 Todo 进度/结果
- **Subscribe**：Agent 接收通知（新任务、状态变更等），后端发布
- **Request-Reply**：Agent 查询数据（任务详情、待办列表等），后端响应
- REST API 只服务前端 UI
- Agent 侧只需一个 NATS 连接，实现更简单、更解耦
- 一切通信经过后端中转，保证数据一致性和可追溯性
- 本节中的 `task.create`、`todo.assigned` 等均为动作简称，完整 subject 见 [message-protocol.md](./message-protocol.md)

### PM Agent 角色

- 每个项目绑定一个 PM Agent，负责需求沟通 → 任务规划 → 指派分发 → 结果汇总
- 用户不直接创建任务，而是通过与 PM Agent 对话来驱动
- PM Agent 通过 NATS 的 `task.create` 消息创建任务并拆分 Todo

### Todo 作为最小执行单元

- MVP 中所有任务都包含 Todo 列表
- 简单任务用一个 Todo 表示，复杂任务用多个 Todo 表示
- 执行 Agent 只处理 Todo，Task 状态由服务端聚合

### 指派策略

每个 Todo 显式绑定一个执行 Agent：`assignee_agent_id` + `assignee_node_id`。服务端按 Todo 分派并校验回传身份。

### Agent 管理策略

- 用户先在平台内按 `node_id` 添加 Agent，建立平台记录与 NATS 节点的映射。
- `node_id` 是 Agent 的稳定身份标识，用于 NATS 收发和服务端鉴权。
- 用户可编辑 Agent 的展示与能力信息：名称、role、描述、capabilities、config。
- `capabilities` 为 PM Agent 规划任务时提供候选匹配信息。
- `role=pm` 表示该 Agent 可作为项目经理 Agent。
- Agent 在线状态由心跳驱动，项目 PM 不在线时禁止用户发起需求。

### Agent 是外部进程

后端不内置 Agent 运行时。Agent 可以是任何语言实现、运行在任何地方。平台只负责任务编排，不负责 Agent 执行。
