# 生产环境硬件资源评估报告

本报告基于 ClawSynapse 网络 + TrustMesh Workspace 架构，评估支撑 **百万级用户、万级并发** 所需的服务器资源。

## 一、系统拓扑

```
                ┌──────────────────────────────────┐
                │      NATS Cluster (3~5 节点)      │
                │      共享消息总线基础设施            │
                └──────────┬───────────────────────┘
                           │
      ┌────────────────────┼────────────────────┐
      │                    │                    │
      ▼                    ▼                    ▼
┌───────────┐       ┌───────────┐        ┌───────────┐
│Workspace A│       │Workspace B│  ...   │Workspace N│
│(TrustMesh)│       │(TrustMesh)│        │(TrustMesh)│
│           │       │           │        │           │
│ Backend   │       │ Backend   │        │ Backend   │
│ Frontend  │       │ Frontend  │        │ Frontend  │
│ MongoDB   │       │ MongoDB   │        │ MongoDB   │
│ Qdrant    │       │ Qdrant    │        │ Qdrant    │
│clawsynapsed│      │clawsynapsed│       │clawsynapsed│
└─────┬─────┘       └─────┬─────┘        └─────┬─────┘
      │                    │                    │
      ▼                    ▼                    ▼
┌───────────┐       ┌───────────┐        ┌───────────┐
│ Agent 节点│       │ Agent 节点│        │ Agent 节点│
│clawsynapsed│      │clawsynapsed│       │clawsynapsed│
│+ openclaw │       │+ openclaw │        │+ openclaw │
└───────────┘       └───────────┘        └───────────┘
```

**关键特征**：
- 每个 Workspace 独立运行完整的 TrustMesh 服务栈
- Agent 节点可被多个 Workspace 共享（通过 ClawSynapse 网络发现）
- 所有节点通过 NATS 集群互联
- 水平扩展通过增加 Workspace 节点实现，而非集群化单个 Workspace

## 二、目标规模

| 指标 | 数值 |
|---|---|
| 注册用户 | 1,000,000 |
| DAU | ~100,000 |
| Workspace 节点 | 100~500（每个服务 2,000~10,000 用户） |
| Agent 节点 | 2,000~10,000（PM + 执行 Agent） |
| 全网峰值并发 | 10,000 |
| 日均 ClawSynapse 消息 | ~5,000,000 条 |
| 全网公网带宽 | ~2~3 TB/月 |

## 三、单节点资源需求

### Workspace 节点（TrustMesh 实例）

每个 Workspace 运行 5 个 Docker 容器：Backend、clawsynapsed、Frontend(Nginx)、MongoDB、Qdrant。

| 规模 | 推荐规格 | 用户容量 |
|---|---|---|
| 小型 | 4C8G + 100GB SSD | < 1,000 用户 |
| 中型 | 8C16G + 200GB SSD | 1,000~5,000 用户 |
| 大型 | 8C32G + 500GB SSD | 5,000~10,000 用户 |

### Agent 节点（clawsynapsed + OpenClaw）

| 状态 | 规格 | 说明 |
|---|---|---|
| 活跃 | 1C2G + 10GB SSD | 正在执行任务（主要 I/O 等待 LLM API，CPU 利用率低） |
| 冷备 | 0.25C256M + 2GB | 仅 clawsynapsed 维持心跳，按需唤醒 |

> clawsynapsed 本身极轻量（0.5C/256M），OpenClaw Agent 运行时以网络 I/O 为主，对 CPU 需求不高。

### NATS 节点

| 规格 | 说明 |
|---|---|
| 4C8G + 100GB SSD | 单节点可处理 10M+ msg/s，远超业务需求 |

## 四、服务器总量

### K8s 容器化部署方案（推荐）

所有节点均以容器运行在 K8s 集群中，由调度器统一分配物理资源。

**Workspace 集群资源估算**（假设 200 小型 + 200 中型 + 100 大型 = 500 Workspace）：

| Workspace 规模 | 数量 | 单个资源 | 小计 |
|---|---|---|---|
| 小型 (4C8G) | 200 | 4C8G | 800C / 1.6TB |
| 中型 (8C16G) | 200 | 8C16G | 1,600C / 3.2TB |
| 大型 (8C32G) | 100 | 8C32G | 800C / 3.2TB |
| **合计** | **500** | | **3,200C / 8TB** |

**全网物理机估算**（按 70% 装箱率）：

| 集群用途 | 物理机规格 | 台数 | 推算依据 |
|---|---|---|---|
| Workspace 集群 | 64C128G, 2TB SSD | 85~95 台 | CPU: 3,200C ÷ 64C ÷ 70% ≈ 72 台；内存: 8TB ÷ 128G ÷ 70% ≈ 90 台；**内存为瓶颈** |
| Agent 集群 | 32C64G, 500GB SSD | 35~45 台 | 900 活跃 × 1C2G = 900C/1.8TB；CPU: 900÷32÷70% ≈ 40 台 |
| NATS 集群 | 4C8G, 100GB SSD | 3~5 台 | 独立部署保证消息总线稳定 |
| 监控/网关 | 8C16G, 500GB SSD | 3~5 台 | Prometheus + Grafana + ELK |
| **合计** | | **~130~150 台** | |

> Workspace 集群的瓶颈在内存（MongoDB + Qdrant 吃内存），物理机需选大内存型号。Agent 主要等待 LLM API 响应（I/O 密集），CPU 利用率通常 < 20%，实际密度可高于理论值。

## 五、分阶段部署

| 阶段 | 用户规模 | Workspace | Agent | 服务器数（K8s） |
|---|---|---|---|---|
| MVP 验证 | 1,000 | 1~3 | 10~20 | 1~2 台 |
| 小规模商用 | 10,000 | 5~20 | 50~100 | 5~10 台 |
| 中等规模 | 100,000 | 30~100 | 300~1,000 | 30~55 台 |
| 大规模运营 | 1,000,000 | 100~500 | 2,000~10,000 | 130~150 台 |

## 六、核心结论

1. **K8s 部署 ~130~150 台物理机** 可支撑百万用户（Workspace 用 64C128G，Agent 用 32C64G）。
2. **Agent 节点数量多但极轻量**（I/O 密集型），适合高密度容器化混部。
3. **Workspace 单机可跑**：中型 8C16G 即可服务 3,000~5,000 用户。
4. **ClawSynapse 网络层几乎不占服务器**：NATS 3~5 台 + clawsynapsed 作为 sidecar 零额外服务器。
5. **MVP 阶段 1~2 台机器即可启动**。

## 七、优化方向

- **Workspace**：K8s 统一调度 + MongoDB/Qdrant 共享集群（按 database/collection 隔离）
- **Agent**：冷热分离 + 共享 Agent 池 + 按任务队列自动伸缩
- **NATS**：跨地域用 Leafnode 架构 + 大消息体压缩
- **数据层**：大型 Workspace 从"内存优先"升级为"MongoDB + Redis 缓存"；历史数据归档到对象存储
