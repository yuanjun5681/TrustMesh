# API 设计

## 通信架构总览

```
┌──────────┐  REST API   ┌──────────────┐  NATS Pub/Sub  ┌──────────┐
│  前端 UI  │ ──────────→ │  TrustMesh   │ ←────────────→ │  Agent   │
│ (React)  │ ←────────── │   后端 (Gin)  │                │ (任意语言) │
└──────────┘  JSON       └──────┬───────┘                └──────────┘
                                │
                           MongoDB
```

- **前端 ↔ 后端**：REST API（JSON），JWT 认证
- **Agent ↔ 后端**：全部通过 NATS 消息通信，Agent 不调用 HTTP 接口
- 后端同时是 NATS 的发布者和订阅者

## 一、REST API（前端使用）

### 认证方式

`Authorization: Bearer <JWT>`

### 认证

```
POST   /api/v1/auth/register          注册
POST   /api/v1/auth/login             登录（返回 JWT）
```

### 项目

```
POST   /api/v1/projects               创建项目（自动绑定 PM Agent）
GET    /api/v1/projects               列出我的项目
GET    /api/v1/projects/:id           项目详情（含 PM Agent 信息）
PATCH  /api/v1/projects/:id           更新项目
DELETE /api/v1/projects/:id           归档项目
```

约束：
- 项目绑定的 `pm_agent_id` 必须指向一个 `role=pm` 的 Agent。
- 如果 PM Agent 当前不在线，不允许创建新对话或发送需求消息。

### 对话（用户 ↔ PM Agent）

```
POST   /api/v1/projects/:projectId/conversations          发起新对话
GET    /api/v1/projects/:projectId/conversations           列出对话
GET    /api/v1/conversations/:id                           对话详情（含消息历史）
POST   /api/v1/conversations/:id/messages                  发送消息（用户 → PM Agent）
```

> 用户发送消息后，后端写入 DB，然后通过 NATS 发布 `notify.{pmNodeId}.conversation.message` 通知 PM Agent。

发送前校验：
- 项目必须已绑定 PM Agent。
- 该 PM Agent 的 `role` 必须为 `pm`。
- 该 PM Agent 必须处于在线状态；否则返回业务错误，例如 `PM_AGENT_OFFLINE`。

### 任务（用户侧只读）

```
GET    /api/v1/projects/:projectId/tasks    列出任务（支持 ?status= 筛选）
GET    /api/v1/tasks/:id                     任务详情（含 Todos）
GET    /api/v1/tasks/:id/events              获取事件流
```

> MVP 中任务由 PM Agent 创建，人类用户通过 UI 查看状态和结果，不直接通过 REST 创建或拆分任务。

### Agent 管理（人类用户操作）

```
POST   /api/v1/agents                 添加 Agent（按 node_id 绑定）
GET    /api/v1/agents                 列出我的 Agent
GET    /api/v1/agents/:id             Agent 详情
PATCH  /api/v1/agents/:id             更新 Agent 信息
DELETE /api/v1/agents/:id             删除 Agent
```

`POST /api/v1/agents` 请求体示例：

```json
{
  "node_id": "node-dev-001",
  "name": "Backend Agent A",
  "role": "developer",
  "description": "负责后端接口与数据库实现",
  "capabilities": ["backend", "golang", "mongodb"],
  "type": "llm",
  "config": {
    "model": "gpt-5",
    "system_prompt": "You are a backend engineer",
    "tools": ["repo", "terminal"],
    "temperature": 0.2
  }
}
```

`PATCH /api/v1/agents/:id` 可更新字段：

```json
{
  "name": "Backend Agent Alpha",
  "role": "reviewer",
  "description": "负责代码评审与质量检查",
  "capabilities": ["backend", "review", "security"],
  "config": {
    "model": "gpt-5",
    "system_prompt": "You are a reviewer",
    "tools": ["repo", "terminal"],
    "temperature": 0.1
  }
}
```

约束：
- `node_id` 必填且唯一，服务端用于把 `agent.{nodeId}.>` 主题映射到 Agent。
- `PATCH` 默认不允许修改 `node_id`。如后续需要迁移节点，应单独设计“换绑节点”接口。
- `role` 用于控制 PM 与执行 Agent 的权限边界。
- `capabilities` 用于 PM Agent 规划 Todo 时筛选候选执行 Agent。

## 二、NATS 消息协议（Agent 使用）

详细协议已拆分到 [message-protocol.md](./message-protocol.md)，该文档是唯一的 subject 和 payload 规范来源。

本文件只保留使用约定：

- Agent 发布业务动作：`agent.{nodeId}.{domain}.{action}`
- 服务器推送通知：`notify.{nodeId}.{domain}.{action}`
- Agent 发起 RPC：`rpc.{nodeId}.{domain}.{action}`
- Agent 身份由 subject 中的 `nodeId` 映射到平台内 Agent 记录
- PM 权限由 Agent 的 `role=pm` 判定
- Agent 在线状态由 `agent.{nodeId}.system.heartbeat` 维护

## 三、后端 NATS 处理流程

### 典型流程：最小闭环

```
1. 用户通过 REST 发送需求消息。
2. 后端先校验项目绑定的 PM Agent 为 `role=pm` 且当前在线。
3. 校验通过后，后端写入 Conversation，并发布 → notify.{pmNodeId}.conversation.message。
4. PM Agent 分析需求后，发布 → agent.{pmNodeId}.task.create。
5. 后端校验 role=pm，写入 Task + Todos。
6. 后端遍历 todos，逐个发布 → notify.{assigneeNodeId}.todo.assigned。
7. 执行 Agent 收到通知，必要时通过 rpc.{assigneeNodeId}.task.get 拉取任务详情。
8. 执行 Agent 通过 `todo.progress` / `todo.complete` / `todo.fail` 回传执行结果。
9. 后端更新 Todo 状态，并聚合更新 Task 状态。
10. 当全部 Todo 完成时，Task 自动完成；如有 Todo 失败且未恢复，则 Task 标记失败。
```

说明：
- 具体订阅 subject、消息结构和权限矩阵以 [message-protocol.md](./message-protocol.md) 为准。
- 本文件仅定义 REST 入口如何接入这条消息链路。
