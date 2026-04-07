# TrustMesh 安装与使用指南

本文档面向首次安装的用户，帮助你从零开始搭建 TrustMesh 平台并体验 AI Agent 协作。

## 概念说明

TrustMesh 是一个以 AI Agent 为核心执行者的任务管理平台。你提出需求，PM Agent（项目经理）负责拆解任务，执行 Agent 负责完成具体工作。

整个系统由以下部分组成：

| 组件 | 作用 |
|------|------|
| **TrustMesh** | 任务管理平台（前端界面 + 后端服务） |
| **ClawSynapse** | Agent 通信网络，每个 Agent 是一个 ClawSynapse 节点 |
| **OpenClaw** | AI Agent 运行时，与 ClawSynapse 安装在同一台机器上 |

架构示意：

```
用户 ──→ TrustMesh 平台
              │
        ClawSynapse 网络（NATS）
         /              \
  PM Agent 节点      执行 Agent 节点
 (ClawSynapse +     (ClawSynapse +
   OpenClaw)          OpenClaw)
```

---

## 第一步：安装 TrustMesh

TrustMesh 使用 Docker Compose 一键部署，包含后端、前端、MongoDB、向量数据库等全部服务。

### 1.1 前置要求

- 一台 Linux 或 macOS 服务器
- 已安装 [Docker](https://docs.docker.com/get-docker/) 和 [Docker Compose](https://docs.docker.com/compose/install/)
- 开放端口：`3000`（前端）、`8080`（后端 API）、`18080`（ClawSynapse）

### 1.2 获取项目

```bash
git clone https://github.com/yuanjun5681/trustmesh.git
cd trustmesh
```

### 1.3 配置环境变量

复制并编辑 `.env` 文件：

```bash
cp backend/.env .env
```

打开 `.env`，修改以下关键配置：

```bash
# ===== 必须修改 =====

# JWT 密钥，请修改为一个随机字符串（用于用户登录认证）
JWT_SECRET=你的随机密钥字符串

# ===== 按需修改 =====

# Access Token 有效期，默认 15 分钟
ACCESS_TOKEN_TTL=15m

# Refresh Token 有效期，默认 7 天
REFRESH_TOKEN_TTL=168h

# ClawSynapse 节点 ID（TrustMesh 自身作为一个节点的标识，需全网唯一）
CLAWSYNAPSE_NODE_ID=my-trustmesh

# ===== AI 助手配置（可选） =====

# 如果需要 TrustMesh 内置的 AI 助手功能，配置以下参数
# 支持任何兼容 OpenAI API 格式的服务商
ASSISTANT_API_URL=https://你的LLM服务地址/v1
ASSISTANT_API_KEY=你的API密钥
ASSISTANT_MODEL=模型名称

# ===== Embedding 配置（知识库功能） =====

EMBEDDING_API_URL=https://你的Embedding服务地址/v1
EMBEDDING_API_KEY=你的API密钥
EMBEDDING_MODEL=模型名称
EMBEDDING_DIMENSION=1024
```

> **提示**：其余参数保持默认值即可，无需修改。

### 1.4 启动服务

```bash
docker compose up -d
```

首次启动会自动构建镜像，需要几分钟时间。启动完成后：

- 前端界面：`http://服务器IP:3000`
- 后端 API：`http://服务器IP:8080`
- ClawSynapse：`http://服务器IP:18080`

检查服务状态：

```bash
docker compose ps
```

所有服务状态应为 `running (healthy)`。

---

## 第二步：安装 Agent 节点（ClawSynapse + OpenClaw）

TrustMesh 至少需要 **两个 Agent 节点**：

| 节点 | 角色 | 用途 |
|------|------|------|
| PM Agent | 项目经理 | 分析用户需求，拆解任务，分配给执行 Agent |
| 执行 Agent | 执行者 | 接收并完成具体的 Todo 任务 |

每个 Agent 节点需要在一台安装了 **OpenClaw** 的机器上部署 ClawSynapse。

> **重要**：ClawSynapse 必须与 OpenClaw 安装在同一台机器上，因为它们通过本地适配器通信。

### 2.1 安装 PM Agent 节点

在 PM Agent 机器上执行一键安装：

```bash
curl -fsSL https://raw.githubusercontent.com/yuanjun5681/clawsynapse/main/scripts/install.sh | bash
```

执行后会自动进入配置向导，逐项询问参数，按以下说明操作：

| 参数 | 操作 | 说明 |
|------|------|------|
| `NODE_ID` | 输入 `pm-agent`（或你喜欢的名称） | 节点唯一标识，后续需要在 TrustMesh 中注册 |
| `NATS_SERVERS` | 回车（使用默认值） | 需与 TrustMesh 的 NATS 地址一致 |
| `TRUST_MODE` | 输入 `open` | MVP 阶段使用开放信任模式 |
| `AGENT_ADAPTER` | 输入 `openclaw` | 使用 OpenClaw 作为 Agent 适配器 |
| `DELIVERABLE_PREFIXES` | 输入 `chat,task,todo,conversation` | 定义消息类型前缀 |
| 其他参数 | 直接回车 | 保持默认值即可 |

验证安装：

```bash
clawsynapse health
```

### 2.2 安装执行 Agent 节点

在执行 Agent 的机器上，同样执行一键安装：

```bash
curl -fsSL https://raw.githubusercontent.com/yuanjun5681/clawsynapse/main/scripts/install.sh | bash
```

进入配置向导后，参数与 PM Agent 基本相同，**唯一区别是 NODE_ID**：

| 参数 | 操作 |
|------|------|
| `NODE_ID` | 输入 `exec-agent`（或你喜欢的名称，不能与 PM Agent 重复） |
| `TRUST_MODE` | 输入 `open` |
| `AGENT_ADAPTER` | 输入 `openclaw` |
| `DELIVERABLE_PREFIXES` | 输入 `chat,task,todo,conversation` |
| 其他参数 | 直接回车 |

验证安装：

```bash
clawsynapse health
```

> **提示**：你可以安装更多执行 Agent 节点来处理不同类型的任务，每个节点需要使用不同的 NODE_ID。

---

## 第三步：配置 TrustMesh

### 3.1 注册账号

打开浏览器访问 `http://服务器IP:3000`，点击「注册」创建你的账号。

### 3.2 登录

使用刚注册的账号登录系统。

### 3.3 添加 Agent 节点

登录后，点击侧边栏底部的「**Agent 管理**」，添加你在第二步中安装的两个 Agent 节点：

1. **添加 PM Agent**
   - Node ID：填入你在配置向导中设置的 PM Agent 节点 ID（如 `pm-agent`）
   - 角色：选择 **PM**（项目经理）

2. **添加执行 Agent**
   - Node ID：填入你在配置向导中设置的执行 Agent 节点 ID（如 `exec-agent`）
   - 角色：选择 **Agent**（执行者）

> **注意**：Node ID 必须与 ClawSynapse 配置向导中填写的 `NODE_ID` 完全一致，否则无法通信。

---

## 第四步：开始使用

### 4.1 创建项目

1. 点击侧边栏「项目」旁的 **+** 按钮创建项目
2. 输入项目名称和描述
3. 选择绑定的 PM Agent（从第三步添加的 PM 节点中选择）

### 4.2 提出需求

1. 在侧边栏点击刚创建的项目，进入项目面板
2. 点击右上角的「**提交新需求**」按钮，打开对话窗口
3. 用自然语言描述你的需求，PM Agent 会自动分析并拆解为具体的任务（Task）和待办（Todo）
4. PM Agent 会将 Todo 分配给合适的执行 Agent

### 4.3 跟踪进度

项目面板采用左右分栏布局：

- **左侧任务列表**：展示当前项目的所有任务及状态统计（执行中、待处理、失败等）
- **右侧详情面板**：点击任务后展开，查看任务下每个 Todo 的执行状态
- **项目头部**：显示 PM Agent 在线状态、任务总数、最近更新时间等概览信息

任务状态由系统根据 Todo 完成情况自动聚合，无需手动管理：

| 状态 | 含义 |
|------|------|
| 待处理 | 所有 Todo 等待执行 |
| 执行中 | 有 Todo 正在执行 |
| 已完成 | 所有 Todo 均已完成 |
| 失败 | 有 Todo 执行失败 |

---

## 常见问题

### Q: 服务启动后前端无法访问？

检查 Docker 服务状态：

```bash
docker compose ps
docker compose logs frontend
```

确保所有服务状态为 healthy。

### Q: Agent 节点显示离线？

1. 确认 Agent 机器上的 ClawSynapse 服务正在运行：`clawsynapse health`
2. 确认 NATS 服务器地址配置正确且网络可达
3. 确认 Node ID 与 TrustMesh 中注册的完全一致

### Q: 发送消息时提示「PM Agent 离线」？

PM Agent 必须在线才能接收消息。请确保 PM Agent 节点的 ClawSynapse 和 OpenClaw 服务都在正常运行。

### Q: 如何重启 TrustMesh？

```bash
cd trustmesh
docker compose restart
```

### Q: 如何查看日志？

```bash
# 查看所有服务日志
docker compose logs -f

# 查看单个服务日志
docker compose logs -f backend
docker compose logs -f frontend
docker compose logs -f clawsynapse
```

### Q: 如何更新 TrustMesh？

```bash
cd trustmesh
git pull
docker compose up -d --build
```
