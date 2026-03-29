# TrustMesh Backend

Go 后端服务，负责：

- JWT 认证
- Agent / Project / Conversation / Task API
- `POST /webhook/clawsynapse` 接收 ClawSynapse WebhookAdapter 推送
- 调用 `clawsynapsed` Local API 发送业务消息
- 周期性同步 `GET /v1/peers` 更新 Agent 在线状态

参考文档：

- [docs/mvp-design.md](/Volumes/UWorks/Projects/TrustMesh/docs/mvp-design.md)
- [docs/api-reference.md](/Volumes/UWorks/Projects/TrustMesh/docs/api-reference.md)
- [docs/message-protocol.md](/Volumes/UWorks/Projects/TrustMesh/docs/message-protocol.md)

## 当前架构

后端已经按 ClawSynapse 集成模式编写，不再直接依赖 NATS 客户端：

- 发送消息：调用 `clawsynapsed` Local API `POST /v1/publish`
- 接收消息：由 `clawsynapsed` 的 `WebhookAdapter` 回调 `POST /webhook/clawsynapse`
- 节点发现：通过 `GET /v1/peers` 同步外部 Agent 在线状态

关键模块：

- [backend/internal/clawsynapse/client.go](/Volumes/UWorks/Projects/TrustMesh/backend/internal/clawsynapse/client.go)
- [backend/internal/clawsynapse/webhook.go](/Volumes/UWorks/Projects/TrustMesh/backend/internal/clawsynapse/webhook.go)
- [backend/internal/clawsynapse/sync.go](/Volumes/UWorks/Projects/TrustMesh/backend/internal/clawsynapse/sync.go)
- [backend/internal/app/router.go](/Volumes/UWorks/Projects/TrustMesh/backend/internal/app/router.go)

## 本地开发

启动基础依赖：

```bash
docker compose up -d mongo clawsynapse
```

说明：

- 本仓库不再编排本地 NATS
- `clawsynapse` 默认连接外部 NATS：`nats://220.168.146.21:9414`
- 如需覆盖，可在启动前设置 `NATS_SERVERS`

启动后端：

```bash
cp .env.example .env
go run ./cmd/server
```

默认监听：`http://127.0.0.1:8080`

服务启动时会自动尝试加载 `.env`，也兼容从仓库根目录启动时读取 `backend/.env`。

## Docker Compose 部署

直接在仓库根目录启动整套服务：

```bash
docker compose up -d --build
```

这会启动：

- `mongo`
- `backend`
- `clawsynapse`
- `frontend`

其中后端容器内默认配置为：

- `MONGO_URI=mongodb://mongo:27017`
- `CLAWSYNAPSE_API_URL=http://clawsynapse:18080`
- TrustMesh 本地节点身份通过 `GET /v1/health` 动态读取，不再单独配置 nodeId

其中 `clawsynapse` 容器默认配置为：

- `NATS_SERVERS=nats://220.168.146.21:9414`
- `DELIVERABLE_PREFIXES=chat,task,todo,conversation`
- `TRANSFER_DIR=/var/lib/trustmesh-transfers`

Artifact 文件访问方式：

- `clawsynapse` 将接收到的文件写入共享卷 `trustmesh-transfer-data`
- `backend` 以只读方式挂载同一目录 `/var/lib/trustmesh-transfers`
- 后端 artifact 内容接口继续本地打开 `localPath`，但这个路径现在属于共享卷，而不是 `clawsynapse` 私有文件系统

若历史 volume 已按错误权限创建，重建共享卷：

```bash
docker compose down
docker volume rm trustmesh_trustmesh-transfer-data
docker compose up -d --build
```

知识库文件使用独立卷 `trustmesh-knowledge-data:/var/lib/trustmesh-knowledge`。后端镜像启动时会先修正该目录的所有权，再以非 root 用户运行 `trustmesh-server`，用来避免以下生产报错：

```text
mkdir /var/lib/trustmesh-knowledge/...: permission denied
```

线上已存在错误权限卷时，通常只需要重新构建并重启后端：

```bash
docker compose up -d --build backend
```

如果需要彻底清空知识库文件，再删除卷重建：

```bash
docker compose down
docker volume rm trustmesh_trustmesh-knowledge-data
docker compose up -d --build backend
```

## 关键环境变量

- `PORT`，默认 `8080`
- `JWT_SECRET`，默认 `trustmesh-dev-secret`
- `TOKEN_TTL`，默认 `24h`
- `LOG_LEVEL`，默认 `info`
- `ALLOW_ALL_CORS`，默认 `true`
- `MONGO_ENABLED`，默认 `true`
- `MONGO_URI`，默认 `mongodb://127.0.0.1:27017`
- `MONGO_DATABASE`，默认 `trustmesh`
- `MONGO_TIMEOUT`，默认 `5s`
- `CLAWSYNAPSE_API_URL`，默认 `http://127.0.0.1:18080`
- `CLAWSYNAPSE_TIMEOUT`，默认 `3s`
- `CLAWSYNAPSE_PEER_SYNC_INTERVAL`，默认 `10s`

## 健康检查

```bash
curl http://127.0.0.1:8080/healthz
```

## 烟雾测试

服务启动后，可以执行最小闭环脚本：

```bash
bash ./scripts/smoke-task-flow.sh
```

或：

```bash
bash ./scripts/smoke-progress-fail-flow.sh
```
