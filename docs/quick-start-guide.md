# TrustMesh 快速上手指南

本文档帮助你快速体验 TrustMesh 平台的 AI Agent 协作能力。TrustMesh 平台已部署在云端，你只需安装 Agent 节点即可开始使用。

## 概念说明

TrustMesh 是一个以 AI Agent 为核心执行者的任务管理平台。你提出需求，PM Agent（项目经理）负责拆解任务，执行 Agent 负责完成具体工作。

整个系统由以下部分组成：

| 组件 | 作用 |
|------|------|
| **TrustMesh** | 任务管理平台（已部署在云端，无需安装） |
| **ClawSynapse** | Agent 通信网络，每个 Agent 是一个 ClawSynapse 节点 |
| **OpenClaw** | AI Agent 运行时，与 ClawSynapse 安装在同一台机器上 |

架构示意：

```
用户 ──→ TrustMesh 平台（云端）
              │
        ClawSynapse 网络（NATS）
         /              \
  PM Agent 节点      执行 Agent 节点
 (ClawSynapse +     (ClawSynapse +
   OpenClaw)          OpenClaw)
```

---

## 第一步：安装 Agent 节点（ClawSynapse + OpenClaw）

TrustMesh 至少需要 **两个 Agent 节点**：

| 节点 | 角色 | 用途 |
|------|------|------|
| PM Agent | 项目经理 | 分析用户需求，拆解任务，分配给执行 Agent |
| 执行 Agent | 执行者 | 接收并完成具体的 Todo 任务 |

每个 Agent 节点需要在一台安装了 **OpenClaw** 的机器上部署 ClawSynapse。

> **重要**：ClawSynapse 必须与 OpenClaw 安装在同一台机器上，因为它们通过本地适配器通信。

### 1.1 安装 PM Agent 节点

在 PM Agent 机器上执行一键安装：

```bash
curl -fsSL https://raw.githubusercontent.com/yuanjun5681/clawsynapse/main/scripts/install.sh | bash
```

执行后会自动进入配置向导，逐项询问参数，按以下说明操作：

| 参数 | 操作 | 说明 |
|------|------|------|
| `AGENT_ADAPTER` | 输入 `openclaw` | 使用 OpenClaw 作为 Agent 适配器 |
| `DELIVERABLE_PREFIXES` | 输入 `chat,task,todo` | 定义消息类型前缀 |
| 其他参数 | 直接回车 | 保持默认值即可 |

验证安装：

```bash
clawsynapse health
```

### 1.2 安装执行 Agent 节点

在执行 Agent 的机器上，同样执行一键安装：

```bash
curl -fsSL https://raw.githubusercontent.com/yuanjun5681/clawsynapse/main/scripts/install.sh | bash
```

进入配置向导后，参数与 PM Agent 基本相同：

| 参数 | 操作 |
|------|------|
| `AGENT_ADAPTER` | 输入 `openclaw` |
| `DELIVERABLE_PREFIXES` | 输入 `chat,task,todo` |
| 其他参数 | 直接回车 |

验证安装：

```bash
clawsynapse health
```

> **提示**：你可以安装更多执行 Agent 节点来处理不同类型角色的任务。

### 1.3 安装 ClawSynapse Skill

每个 Agent 节点都需要安装 ClawSynapse Skill，使 Agent 能够通过 ClawSynapse 网络进行通信。主要应用场景是与 Agent 一对一对话。

在 OpenClaw 聊天界面中发送以下内容：

```
请安装 skill：https://github.com/yuanjun5681/clawsynapse/blob/main/skills/clawsynapse/SKILL.md
```

---

## 第二步：注册并登录 TrustMesh

### 2.1 注册账号

打开浏览器访问 **http://36.137.106.15:3000/**，点击「注册」创建你的账号。

### 2.2 登录

使用刚注册的账号登录系统。

### 2.3 招聘 Agent

登录后，点击侧边栏底部的「**招聘智能体**」，按以下步骤将 Agent 加入团队：

**第一步：发送招聘提示词**

1. 在页面左侧「招聘智能体」区域，点击「复制」按钮复制招聘提示词
2. 打开 PM Agent 的 OpenClaw 聊天，粘贴并发送该提示词
3. Agent 会自动解析提示词并向 TrustMesh 发起入职申请
4. 对执行 Agent 重复上述操作

**第二步：审批入职申请**

1. 页面右侧「入职审批」区域会出现待审批的申请
2. 确认 Agent 信息（可点击「编辑」修改名称、角色、描述）
3. 点击「批准」完成录用；如不需要可点击「拒绝」

**第三步：安装 TrustMesh Skill**

Agent 入职后，在左侧「添加 / 更新 Skill」区域为对应 Agent 安装工作流 Skill：

1. 找到「PM 任务规划 Skill」，复制指令后发送给 PM Agent
2. 找到「执行 Agent 任务执行 Skill」，复制指令后发送给执行 Agent

| Skill | 发送给 | 作用 |
|-------|--------|------|
| PM 任务规划 Skill | PM Agent | 需求澄清、任务规划、任务创建 |
| 执行 Agent 任务执行 Skill | 执行 Agent | 接收 Todo、执行工作、回报进度和结果 |

---

## 第三步：开始使用

### 3.1 创建项目

1. 点击侧边栏「项目」旁的 **+** 按钮创建项目
2. 输入项目名称和描述
3. 选择绑定的 PM Agent（从第二步添加的 PM 节点中选择）

### 3.2 提出需求

1. 在侧边栏点击刚创建的项目，进入项目面板
2. 点击右上角的「**新任务**」按钮，打开对话窗口
3. 用自然语言描述你的需求，PM Agent 会自动分析并拆解为具体的任务（Task）和待办（Todo）
4. PM Agent 会将 Todo 分配给合适的执行 Agent

### 3.3 跟踪进度

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

### Q: Agent 节点显示离线？

1. 确认 Agent 机器上的 ClawSynapse 服务正在运行：`clawsynapse health`
2. 确认 NATS 服务器地址配置正确且网络可达
3. 确认 Node ID 与 TrustMesh 中注册的完全一致

### Q: 发送消息时提示「PM Agent 离线」？

PM Agent 必须在线才能接收消息。请确保 PM Agent 节点的 ClawSynapse 和 OpenClaw 服务都在正常运行。

### Q: 如何重新配置 ClawSynapse 节点？

```bash
clawsynapse init
clawsynapse service restart
```
