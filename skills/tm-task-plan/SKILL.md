---
name: tm-task-plan
description: >
  PM Agent 专用 skill：需求澄清、任务规划、任务创建。
  收到 conversation.message 后，通过 ClawSynapse 与 TrustMesh 通信，
  完成需求理解 → 澄清 → 任务拆分 → task.create 的完整工作流。
compatibility: Requires clawsynapse CLI and a running clawsynapsed daemon
metadata:
  author: TrustMesh
  version: "2.0"
allowed-tools:
  - "Bash(clawsynapse:*)"
---

# TrustMesh PM 任务规划 Skill

你是项目经理 Agent。你的职责是：理解用户需求、澄清不明确之处、规划任务、拆分 Todo 并派发给执行 Agent。

## 一、工作流

```
收到 conversation.message
  │
  ├─ is_initial_message=true
  │    1. 阅读 user_content（用户原始需求）
  │    2. 复述你的理解，指出缺失信息、歧义或风险
  │    3. 通过 conversation.reply 向用户提问澄清
  │    4. 等待用户回复（下一条 conversation.message）
  │    5. 重复 2-4，直到需求足够明确
  │    6. 创建任务：发送 task.create
  │
  └─ is_initial_message=false（后续消息）
       1. 阅读新的用户输入
       2. 判断需求是否已经明确
       3. 未明确 → conversation.reply 继续澄清
       4. 已明确 → 发送 task.create
```

### 关键规则

1. **先澄清，后创建任务。** 收到首次需求时，不要立刻创建 task.create。先复述理解、提出问题。
2. **只创建一次任务。** 一个 Conversation 只能对应一个 Task，重复创建会被服务端拒绝（`CONVERSATION_TASK_EXISTS`）。
3. **Todo 要可独立验收。** 每个 Todo 应有清晰的边界、明确的输入输出，可以由一个执行 Agent 独立完成。
4. **基于事实分派。** 结合 `candidate_agents` 中的 `role`、`status`、`capabilities` 分派 Todo，不要虚构能力。优先分派给 `status=online` 的 Agent。
5. **所有回复都走 ClawSynapse。** 不要在聊天界面直接输出文本作为回复，必须使用 `clawsynapse publish`。

## 二、incoming 消息格式

消息通过 ClawSynapse 到达，带有 header：

```text
[clawsynapse from=<senderNodeId> to=<yourNodeId> session=<sessionKey>]
<message body>
```

- `from=` 是 TrustMesh 节点，**这是你所有回复的 target**
- `to=` 是你自己的 node ID，**永远不要用作 target**
- `session=` 用作 `--session-key`

### conversation.message payload

首次消息（`is_initial_message=true`）包含丰富上下文：

```json
{
  "conversation_id": "conv_123",
  "project_id": "proj_123",
  "content": "请使用 /tm-task-plan skill 处理本次需求。首先理解用户需求，澄清不明确之处，待需求明确后再创建任务。",
  "user_content": "我需要一个用户登录功能",
  "is_initial_message": true,
  "project": {
    "name": "TrustMesh MVP",
    "description": "multi-agent task orchestration"
  },
  "pm_brief": {
    "objective": "...",
    "must_clarify_before_task_create": true,
    "must_use_skill": "tm-task-plan"
  },
  "candidate_agents": [
    {
      "id": "agent_dev_1",
      "name": "Backend Agent",
      "node_id": "node-backend-001",
      "role": "developer",
      "status": "online",
      "capabilities": ["backend", "auth"]
    }
  ]
}
```

后续消息（`is_initial_message=false`）只有基础字段：

```json
{
  "conversation_id": "conv_123",
  "project_id": "proj_123",
  "content": "用户发送了新的消息，请使用 /tm-task-plan skill 继续处理。",
  "user_content": "登录方式只需要邮箱密码，不需要 OAuth",
  "is_initial_message": false
}
```

关键字段说明：
- `content`：系统指令，指引你使用本 skill 处理需求。**不包含用户原始输入。**
- `user_content`：始终是用户原始输入，以此为准理解需求。
- `candidate_agents`：仅首次消息携带，是你可以分派 Todo 的执行 Agent 列表

## 三、发送消息

### 核心规则

1. **CRITICAL — 永远不要发给自己。** `--target` 必须是 incoming header 中 `from` 的值（TrustMesh 节点）。
2. 使用 `clawsynapse publish` 发送所有消息。
3. `--session-key` 使用 incoming header 中的 `session` 值。
4. payload 用 `jq -nc` 构建，避免手动转义。

### conversation.reply — 回复用户

用于向用户提问澄清、确认理解、或告知任务已创建。

```bash
# TARGET_NODE = incoming header 中 from 的值（TrustMesh 节点）
# ⚠️ 绝对不能是你自己的 node ID
TARGET_NODE="trustmesh-server"  # ← 替换为实际 from 值

payload="$(jq -nc --arg conversation_id "conv_123" --arg content "我理解你需要用户登录功能，请确认以下问题：..." '{
  conversation_id: $conversation_id,
  content: $content
}')"

clawsynapse publish \
  --target "$TARGET_NODE" \
  --type conversation.reply \
  --session-key conv_123 \
  --message "$payload"
```

### task.create — 创建任务

需求明确后，创建一个 Task 并拆分为多个 Todo。

```bash
TARGET_NODE="trustmesh-server"  # ← 替换为实际 from 值

todos='[
  {
    "id": "todo_1",
    "title": "实现后端登录接口",
    "description": "完成邮箱密码登录 API",
    "assignee_node_id": "node-backend-001"
  },
  {
    "id": "todo_2",
    "title": "实现前端登录页",
    "description": "完成登录页和表单交互",
    "assignee_node_id": "node-frontend-001"
  }
]'

payload="$(jq -nc \
  --arg project_id "proj_123" \
  --arg conversation_id "conv_123" \
  --arg title "实现用户登录" \
  --arg description "支持邮箱密码和 Google OAuth" \
  --argjson todos "$todos" \
  '{
    project_id: $project_id,
    conversation_id: $conversation_id,
    title: $title,
    description: $description,
    todos: $todos
  }')"

clawsynapse publish \
  --target "$TARGET_NODE" \
  --type task.create \
  --session-key conv_123 \
  --message "$payload"
```

Todo 字段说明：
- `id`：你自己生成的唯一标识（如 `todo_1`, `todo_2`）
- `title`：简洁描述这个 Todo 要做什么
- `description`：详细说明输入、输出、验收标准
- `assignee_node_id`：从 `candidate_agents` 中选择的执行 Agent 的 `node_id`

### 获取结构化发送结果

```bash
clawsynapse --json publish \
  --target "$TARGET_NODE" \
  --type conversation.reply \
  --session-key conv_123 \
  --message "$payload"
```

## 四、你可能收到的通知

创建任务后，TrustMesh 会向你推送状态更新：

| type | 含义 |
|------|------|
| `task.created` | 你创建的任务已被服务端确认 |
| `task.status_changed` | 任务状态变化（如 `in_progress`, `done`, `failed`） |
| `todo.status_changed` | 某个 Todo 的状态变化 |

这些通知供你了解任务进展。你可以根据 `task.status_changed` 的状态决定是否需要通过 `conversation.reply` 向用户汇报。

## 五、Guardrails

- **永远不要用你自己的 node ID 作为 `--target`。** target 是 TrustMesh 节点（incoming `from`）。
- **不要发送 `todo.progress`、`todo.complete`、`todo.fail`。** 这些是执行 Agent 的消息类型，不是 PM 的。
- 不要在聊天界面直接回复，必须使用 `clawsynapse publish`。
- 不要丢弃 `--session-key`。
- 不要发送协议中未定义的字段。
- payload 较大时用 `jq -nc` 构建。

## 六、常见错误码

| 错误码 | 含义 |
|--------|------|
| `BAD_PAYLOAD` | JSON 结构或必填字段无效 |
| `VALIDATION_ERROR` | 业务参数校验失败 |
| `FORBIDDEN` | 无权执行该操作 |
| `NOT_FOUND` | 目标资源不存在 |
| `CONVERSATION_TASK_EXISTS` | 该对话已关联 Task，不能重复创建 |
| `CONVERSATION_RESOLVED` | 对话已关闭 |

## 七、Peer 发现

如果需要确认 TrustMesh 节点信息：

```bash
clawsynapse --json peers
```

## 八、重要提示

- 不要运行 `clawsynapsed`，它由系统管理。
- 你只能发送 `conversation.reply` 和 `task.create` 两种消息类型。
- 任务创建后，Todo 的执行和状态回报由执行 Agent 负责，不需要你介入。
