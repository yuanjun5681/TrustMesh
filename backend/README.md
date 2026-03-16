# TrustMesh Backend (Basic MVP Service)

基础 Go 后端服务，参考：
- `docs/mvp-design.md`
- `docs/api-reference.md`

## 已实现

- Gin HTTP 服务 + `/api/v1` 路由
- JWT 认证（注册/登录）
- Agent CRUD
- Project CRUD（删除语义为归档）
- Conversation 创建/列表/详情/追加消息
- Task 查询接口（列表/详情/事件）
- 统一错误结构与错误码
- `zap` 日志：启动日志 + 请求日志 + panic recovery
- NATS 最小闭环接入（可选启用）
  - 订阅 `agent.*.*.*`：`heartbeat` / `conversation.reply` / `task.create` / `todo.progress` / `todo.complete` / `todo.fail`
  - 订阅 `rpc.*.*.*`：`task.get` / `todo.assigned` / `project.summary` / `task.by_conversation` / `agent.list`
  - 发布 `notify.*`：`conversation.message` / `task.created` / `task.updated` / `todo.assigned` / `todo.updated`

## 当前实现说明

- 当前版本默认启用 MongoDB 状态持久化（关闭方式：`MONGO_ENABLED=false`）。
- 当前存储实现是 Mongo 分集合持久化 + 内存索引模型。
- 启动时会自动创建基础索引，并从 `users`、`agents`、`projects`、`conversations`、`tasks`、`task_events`、`processed_messages` 集合恢复状态。
- Agent 在线状态由心跳 + 后台超时扫描共同决定，超过 `HEARTBEAT_TTL` 会转为 `offline`。
- `task.create` 和 `todo.complete` 已基于 NATS envelope `id` 做严格幂等去重。
- PM Agent 在线约束已实现（`offline` 时禁止创建/追加对话）。
- 已支持 PM 通过 NATS `agent.{pmNodeId}.task.create` 创建 Task/Todo 并驱动分派。

## 运行

```bash
cp .env.example .env
go run ./cmd/server
```

默认监听：`http://localhost:8080`

说明：服务启动时会自动尝试加载 `.env`（也兼容从仓库根目录启动时读取 `backend/.env`）。
若本地未启动 NATS，可临时设置 `NATS_ENABLED=false` 只跑 REST。
若本地未启动 MongoDB，可临时设置 `MONGO_ENABLED=false` 使用纯内存模式。

## 本地依赖

在仓库根目录启动 MongoDB + NATS：

```bash
docker compose up -d
```

默认端口：
- MongoDB: `127.0.0.1:27017`
- NATS: `127.0.0.1:4222`
- NATS monitoring: `127.0.0.1:8222`

停止：

```bash
docker compose down
```

## 烟雾测试

服务启动后，可以执行最小闭环脚本：

```bash
bash ./scripts/smoke-task-flow.sh
```

该脚本会实际执行：
- REST 注册用户
- REST 创建 PM / Developer Agent
- REST 创建 Project / Conversation
- NATS 发布 `task.create`
- NATS 发布 `todo.complete`
- REST 回查 Task 聚合结果

执行进度/失败态脚本：

```bash
bash ./scripts/smoke-progress-fail-flow.sh
```

该脚本会实际执行：
- REST 注册用户
- REST 创建 PM / Developer Agent
- REST 创建 Project / Conversation
- NATS 发布 `task.create`
- NATS 发布 `todo.progress`
- NATS 发布 `todo.fail`
- REST 回查 Task 状态与事件流

## 关键环境变量

- `PORT`（默认 `8080`）
- `JWT_SECRET`（默认 `trustmesh-dev-secret`）
- `TOKEN_TTL`（默认 `24h`）
- `LOG_LEVEL`（默认 `info`）
- `ALLOW_ALL_CORS`（默认 `true`）
- `HEARTBEAT_TTL`（默认 `30s`）
- `HEARTBEAT_SWEEP_INTERVAL`（默认 `5s`）
- `MONGO_ENABLED`（默认 `true`）
- `MONGO_URI`（默认 `mongodb://127.0.0.1:27017`）
- `MONGO_DATABASE`（默认 `trustmesh`）
- `MONGO_TIMEOUT`（默认 `5s`）
- `NATS_ENABLED`（默认 `true`）
- `NATS_URL`（默认 `nats://127.0.0.1:4222`）
- `NATS_CLIENT_NAME`（默认 `trustmesh-backend`）
- `NATS_TIMEOUT`（默认 `3s`）

## 健康检查

```bash
curl http://localhost:8080/healthz
```

## NATS 启动示例

```bash
NATS_ENABLED=true NATS_URL=nats://127.0.0.1:4222 go run ./cmd/server
```
