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
- NATS 最小闭环接入（旧版，待迁移到 ClawSynapse）

## 架构迁移说明

当前代码中的 NATS 直连实现（`internal/nats/`）将迁移到 ClawSynapse 集成模式：

- **发送消息**：从直连 NATS publish 改为调用 `clawsynapsed` Local API（`POST /v1/publish`）
- **接收消息**：从直连 NATS subscribe 改为 WebhookAdapter 推送到 `POST /webhook/clawsynapse`
- **Agent 在线状态**：从自定义心跳机制改为 ClawSynapse 节点发现（`GET /v1/peers` 同步）
- **消息格式**：不再需要关心 ClawSynapse 内部 Envelope 格式，只需使用 `POST /v1/publish` 的简单请求字段（`targetNode`、`type`、`message`、`metadata`）

迁移后的模块结构：
- `internal/nats/` → `internal/clawsynapse/`
  - `client.go`：clawsynapsed Local API 客户端（`POST /v1/publish`、`GET /v1/peers`）
  - `webhook.go`：WebhookAdapter 消息处理（接收消息）
  - `types.go`：WebhookPayload、PublishRequest 等类型定义
  - `sync.go`：peers 列表定期同步

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

在仓库根目录启动 MongoDB + NATS + clawsynapsed：

```bash
docker compose up -d
```

默认端口：
- MongoDB: `127.0.0.1:27017`
- NATS: `127.0.0.1:4222`
- NATS monitoring: `127.0.0.1:8222`
- clawsynapsed Local API: `127.0.0.1:18080`

停止：

```bash
docker compose down
```

## 烟雾测试

服务启动后，可以执行最小闭环脚本：

```bash
bash ./scripts/smoke-task-flow.sh
```

> 注意：烟雾测试脚本当前仍使用旧版 NATS 直连方式，待 ClawSynapse 迁移完成后更新。

## 关键环境变量

- `PORT`（默认 `8080`）
- `JWT_SECRET`（默认 `trustmesh-dev-secret`）
- `TOKEN_TTL`（默认 `24h`）
- `LOG_LEVEL`（默认 `info`）
- `ALLOW_ALL_CORS`（默认 `true`）
- `MONGO_ENABLED`（默认 `true`）
- `MONGO_URI`（默认 `mongodb://127.0.0.1:27017`）
- `MONGO_DATABASE`（默认 `trustmesh`）
- `MONGO_TIMEOUT`（默认 `5s`）
- `NATS_ENABLED`（默认 `true`）
- `NATS_URL`（默认 `nats://127.0.0.1:4222`）
- `CLAWSYNAPSE_API_URL`（默认 `http://127.0.0.1:18080`，clawsynapsed Local API 地址）
- `CLAWSYNAPSE_NODE_ID`（TrustMesh 节点的 ClawSynapse nodeId）
- `CLAWSYNAPSE_PEER_SYNC_INTERVAL`（默认 `10s`，peers 列表同步间隔）

## 健康检查

```bash
curl http://localhost:8080/healthz
```
