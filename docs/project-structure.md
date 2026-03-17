# 项目结构与实施计划

## 目录结构

```
TrustMesh/
├── README.md
├── docker-compose.yml              # MongoDB + NATS + clawsynapsed
├── Makefile
│
├── backend/
│   ├── cmd/
│   │   └── server/
│   │       └── main.go             # 入口，初始化 Gin + MongoDB + clawsynapsed client
│   ├── internal/
│   │   ├── config/
│   │   │   └── config.go           # 配置加载（env vars）
│   │   ├── model/                  # MongoDB 文档模型
│   │   │   ├── user.go
│   │   │   ├── agent.go
│   │   │   ├── project.go
│   │   │   ├── task.go             # 含 Todo 嵌套结构
│   │   │   ├── task_event.go
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
│   │   │   ├── conversation.go     # 对话 API
│   │   │   └── webhook.go          # ClawSynapse WebhookAdapter 接收端点
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
│   │   ├── clawsynapse/            # ClawSynapse 集成
│   │   │   ├── client.go           # clawsynapsed Local API 客户端（POST /v1/publish）
│   │   │   ├── webhook.go          # WebhookAdapter 消息处理（接收消息）
│   │   │   ├── types.go            # WebhookPayload、PublishRequest 等类型定义
│   │   │   └── sync.go             # GET /v1/peers 定期同步
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
    ├── message-protocol.md         # ClawSynapse 消息传输规范
    ├── agent-engine.md             # Agent 引擎
    ├── frontend-design.md          # 前端设计
    └── project-structure.md        # 本文件
```

## 实施计划

### 阶段 1：基础骨架

- 初始化 Go module（Gin）+ Vite 项目（shadcn/ui）
- docker-compose 配置 MongoDB + NATS + clawsynapsed（WebhookAdapter 模式）
- MongoDB 连接 + 集合初始化 + 索引创建
- ClawSynapse 客户端封装（`internal/clawsynapse/client.go`）
- WebhookAdapter 接收端点（`POST /webhook/clawsynapse`）
- 认证中间件（JWT）
- Agent 管理基础模型与 `node_id` 唯一约束
- peers 列表定期同步，Agent 在线状态判定
- 前端路由 + MainLayout + Sidebar

### 阶段 2：需求进入 PM

- 后端 Project + Conversation API
- 用户发送需求消息前，校验项目 PM Agent 在线（peers 列表可见）
- 用户发送需求消息后，通过 `clawsynapsed` Local API publish `conversation.message` 给 PM Agent
- PM Agent 通过 ClawSynapse 回复 `conversation.reply`，后端通过 WebhookAdapter 接收并写入 DB
- 前端项目列表 + 对话入口 + 基础看板

### 阶段 3：PM 创建任务

- PM Agent 通过 ClawSynapse 发送 `task.create`
- 后端通过 WebhookAdapter 收到 `task.create`，写入 Task + Todos
- 后端按 Todo 的 assignee 通过 Local API publish `todo.assigned`

### 阶段 4：Agent 执行体系

- Agent 注册管理 API（按 `node_id` 添加、编辑名称/role/描述/能力）
- Agent 执行协议（`todo.progress` / `todo.complete` / `todo.fail`），通过 WebhookAdapter 接收
- 通知集成（Todo 指派 → 执行 Agent，执行结果 → 后端 → PM Agent）
- 前端 Agent 管理页 + 任务详情结果展示

### 阶段 5：活动流与打磨

- TaskEvent 写入和查询
- TaskTimeline 组件
- Agent 结果展示 + Todo 进度可视化
- UI 打磨和错误处理

## 关键设计决策

### 基于 ClawSynapse 网络

- TrustMesh 后端不直连 NATS，作为 ClawSynapse 网络中的一个节点
- 发送消息：通过 `clawsynapsed` 的 Local API（`POST /v1/publish`）
- 接收消息：通过 `clawsynapsed` 的 WebhookAdapter 推送到 TrustMesh 的 HTTP webhook 端点
- 所有 Agent（PM Agent、执行 Agent）都是独立的 ClawSynapse 节点
- 通过 ClawSynapse 获得节点发现、身份认证、信任管理能力（MVP 使用 TRUST_MODE=open）
- Agent 在线状态由 ClawSynapse 的 `discovery.announce` 维护，TrustMesh 通过 `GET /v1/peers` 同步

### 选择 MongoDB

- **Schema 灵活**：Task 内嵌 Todos、Agent config、执行 result 等结构天然适合文档模型
- **嵌套文档**：Todos 直接内嵌在 Task 中，无需 JOIN
- **多态存储**：不同类型 Agent 的 config 结构差异大，文档模型更自然
- **开发简单**：无需迁移文件、无需 ORM 映射、Go struct 直接序列化

### 选择 Gin

- 成熟稳定，社区活跃
- 中间件生态丰富（CORS、日志、Recovery）
- 路由分组便于按模块组织（v1/auth、v1/projects、v1/agent）

### PM Agent 角色

- 每个项目绑定一个 PM Agent，负责需求沟通 → 任务规划 → 指派分发 → 结果汇总
- PM Agent 是独立的 ClawSynapse 节点，不内嵌在 TrustMesh 后端中
- 用户不直接创建任务，而是通过与 PM Agent 对话来驱动
- PM Agent 通过 ClawSynapse 发送 `task.create` 消息创建任务并拆分 Todo

### Todo 作为最小执行单元

- MVP 中所有任务都包含 Todo 列表
- 简单任务用一个 Todo 表示，复杂任务用多个 Todo 表示
- 执行 Agent 只处理 Todo，Task 状态由服务端聚合

### 指派策略

每个 Todo 显式绑定一个执行 Agent：`assignee_agent_id` + `assignee_node_id`。TrustMesh 后端按 Todo 分派并校验回传身份。

### Agent 管理策略

- 用户先在平台内按 `node_id` 添加 Agent，建立平台记录与 ClawSynapse 节点的映射。
- `node_id` 对应该 Agent 的 ClawSynapse nodeId，是 Agent 的稳定身份标识。
- 用户可编辑 Agent 的展示与能力信息：名称、role、描述、capabilities。
- `capabilities` 为 PM Agent 规划任务时提供候选匹配信息。
- `role=pm` 表示该 Agent 可作为项目经理 Agent。
- Agent 在线状态由 ClawSynapse 节点发现机制维护，TrustMesh 通过 peers 列表同步。

### Agent 是外部进程

后端不内置 Agent 运行时。Agent 可以是任何语言实现、运行在任何地方，只要运行 `clawsynapsed` 并接入 ClawSynapse 网络即可。平台只负责任务编排，不负责 Agent 执行。
