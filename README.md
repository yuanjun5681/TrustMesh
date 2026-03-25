# TrustMesh

`TrustMesh` 当前是一个前后端分离项目：

- `frontend/`：React + Vite 单页应用
- `backend/`：Go + Gin API 服务，负责认证、项目、会话、任务和 ClawSynapse webhook
- `mongo`：业务数据存储
- 外部 NATS：ClawSynapse 共享消息总线，默认 `nats://220.168.146.21:9414`
- `clawsynapse`：TrustMesh 自身的 ClawSynapse 节点，使用 `webhook` adapter 回调后端

## 项目结构

- [README.md](/Volumes/UWorks/Projects/TrustMesh/README.md)：仓库入口
- [docker-compose.yml](/Volumes/UWorks/Projects/TrustMesh/docker-compose.yml)：完整部署编排
- [backend](/Volumes/UWorks/Projects/TrustMesh/backend)：后端服务
- [frontend](/Volumes/UWorks/Projects/TrustMesh/frontend)：前端应用
- [deploy/clawsynapse](/Volumes/UWorks/Projects/TrustMesh/deploy/clawsynapse)：上游 `clawsynapse` 节点容器构建定义
- [docs](/Volumes/UWorks/Projects/TrustMesh/docs)：架构与协议文档

## Docker Compose 部署

直接启动整套服务：

```bash
docker compose up -d --build
```

默认访问地址：

- 前端: `http://127.0.0.1:3000`
- 后端: `http://127.0.0.1:8080`
- 后端健康检查: `http://127.0.0.1:8080/healthz`
- ClawSynapse Local API: `http://127.0.0.1:18080/v1/health`
- MongoDB: `127.0.0.1:27017`

外部依赖：

- NATS: `nats://220.168.146.21:9414`

停止服务：

```bash
docker compose down
```

删除持久化数据卷：

```bash
docker compose down -v
```

## ClawSynapse 容器说明

Compose 中的 `clawsynapse` 服务会在构建阶段从上游仓库拉取源码并编译 `clawsynapsed`：

- 上游仓库: [yuanjun5681/clawsynapse](https://github.com/yuanjun5681/clawsynapse)
- 默认构建分支: `main`
- 可通过环境变量覆盖：`CLAWSYNAPSE_REF=<tag-or-commit> docker compose build clawsynapse`

当前容器内使用的关键配置：

- `AGENT_ADAPTER=webhook`
- `WEBHOOK_URL=http://backend:8080/webhook/clawsynapse`
- `LOCAL_API_ADDR=0.0.0.0:18080`
- `NATS_SERVERS=nats://220.168.146.21:9414`
- `DELIVERABLE_PREFIXES=chat,task,todo,conversation`
- `TRANSFER_DIR=/var/lib/trustmesh-transfers`

最后这一项是必须的。上游默认前缀是 `chat,task`，不足以把 `conversation.reply` 和 `todo.*` 消息投递给 TrustMesh 后端。

Artifact 文件读取依赖共享传输卷：

- `clawsynapse` 读写 `trustmesh-transfer-data:/var/lib/trustmesh-transfers`
- `backend` 只读挂载同一个卷

这样 `clawsynapse` 返回的 `localPath` 在两个容器里都有效，后端读取 artifact 内容时不会再落到私有容器文件系统。

如果你之前已经创建过旧的 `trustmesh-transfer-data` 卷，升级后建议重建一次相关容器；若仍有权限残留，再执行：

```bash
docker compose down
docker volume rm trustmesh_trustmesh-transfer-data
docker compose up -d --build
```

知识库文件默认写入 `trustmesh-knowledge-data:/var/lib/trustmesh-knowledge`。后端容器启动时会自动校正这个卷的所有权后再降权启动应用，因此生产环境遇到知识库上传报错：

```text
mkdir /var/lib/trustmesh-knowledge/...: permission denied
```

通常只需要重新构建并重启后端：

```bash
docker compose up -d --build backend
```

如果卷权限已被外部手工改坏，仍可选择重建知识库卷：

```bash
docker compose down
docker volume rm trustmesh_trustmesh-knowledge-data
docker compose up -d --build backend
```

如果要覆盖默认值，可以在启动前注入：

```bash
NATS_SERVERS=nats://your-host:4222 \
DELIVERABLE_PREFIXES=chat,task,todo,conversation \
docker compose up -d --build
```

后端本地开发与接口说明见：
- [backend/README.md](/Volumes/UWorks/Projects/TrustMesh/backend/README.md)
