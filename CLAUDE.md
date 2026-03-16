# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 项目概览

TrustMesh 是一个以 AI Agent 为核心执行者的任务编排与项目管理平台。参考 Asana 的项目管理模型，将执行者从人类替换为 AI Agent。

**当前状态**：设计与文档完成阶段，代码实现尚未开始。核心设计文档在 `docs/` 目录下。

## 架构

```
前端 (React) ←→ REST API (JSON, JWT) ←→ 后端 (Go/Gin)
                                              ↕
                                        NATS 消息总线
                                              ↕
                                        Agent (任意语言)
```

- **前端 ↔ 后端**：REST API + JWT 认证
- **Agent ↔ 后端**：NATS Pub/Sub + Request-Reply，Agent 不调用 HTTP
- **Agent 间不直接通信**，一切通过后端中转

### 核心角色

- **用户**：通过 UI 创建项目、提需求、查看状态
- **PM Agent**：每项目绑定一个（role=pm），负责需求分析、任务规划、Todo 拆分
- **执行 Agent**：接收 Todo 并执行，回传进度和结果

### 关键设计决策

- **Todo 是最小执行单元**：Task 包含 Todo 列表，Task 状态由 Todos 自动聚合
- **一个 Conversation 最多对应一个 Task**：PM 对同一 Conversation 只能成功创建一次 Task
- **PM 门禁**：PM 不在线时禁止用户发起需求，返回 PM_AGENT_OFFLINE
- **并发控制**：MongoDB `findOneAndUpdate` + positional operator，每次 Todo 更新递增 `task.version`

## 技术栈

| 层 | 技术 |
|---|---|
| 后端 | Go, Gin, MongoDB (mongo-driver v2), NATS |
| 前端 | React 18, TypeScript, Vite, shadcn/ui (Radix UI), Tailwind CSS v4 |
| 状态管理 | TanStack Query (服务端状态), Zustand (客户端状态) |
| HTTP 客户端 | ky |
| 基础设施 | Docker + docker-compose (MongoDB, NATS) |

## 计划目录结构

```
backend/
  cmd/server/main.go          # 入口点
  internal/
    config/                    # 配置加载
    model/                     # MongoDB 文档模型
    repository/                # 数据访问层
    handler/                   # Gin HTTP handlers
    middleware/                 # JWT、CORS、日志
    service/                   # 业务逻辑、状态机
    nats/                      # NATS 集成 (handler, rpc, publisher)
    dto/                       # 请求/响应 DTO

frontend/
  src/
    components/ui/             # shadcn/ui 组件
    components/{layout,board,task,conversation,agent}/
    api/                       # API 调用封装
    hooks/                     # React hooks
    stores/                    # Zustand stores
    pages/                     # 页面组件
    types/                     # TypeScript 类型
```

## 设计文档索引

| 文档 | 内容 |
|---|---|
| `docs/mvp-design.md` | 总览与功能范围 |
| `docs/data-model.md` | MongoDB 文档结构、约束、索引 |
| `docs/api-design.md` | REST API 端点、NATS 协议设计 |
| `docs/message-protocol.md` | NATS subject、payload、权限矩阵 |
| `docs/agent-engine.md` | PM 工作流、Agent 执行、后端处理 |
| `docs/frontend-design.md` | 页面规划、组件树、交互设计 |
| `docs/project-structure.md` | 目录结构、实施计划、设计决策 |

## NATS 消息命名空间

- `agent.{nodeId}.*` — Agent 发布动作（heartbeat、task.create、todo.progress 等）
- `notify.{nodeId}.*` — 后端推送通知（conversation.message、todo.assigned 等）
- `rpc.{nodeId}.*` — Agent 发起查询（task.get、todo.assigned、agent.list 等）

## Task 状态聚合规则

```
全部 todos pending      → task.pending
存在 todo in_progress   → task.in_progress
全部 todos done         → task.done
存在 todo failed        → task.failed
```
