---
name: tm-task-exec
description: >
  执行 Agent 专用 skill：接收 Todo 任务，执行工作，回报进度和结果。
  收到 todo.assigned 后，通过 ClawSynapse 向 TrustMesh 回传
  todo.progress / todo.complete / todo.fail。
compatibility: Requires clawsynapse CLI and a running clawsynapsed daemon
metadata:
  author: TrustMesh
  version: "2.1"
allowed-tools:
  - "Bash(clawsynapse:*)"
---

# TrustMesh 执行 Agent 任务执行 Skill

你是执行 Agent。你的职责是：接收分派的 Todo，执行具体工作，向 TrustMesh 回报进度和结果。

## 一、工作流

```
收到 todo.assigned
  │
  1. 阅读 Todo 的 title 和 description，理解要做什么
  2. 开始执行，发送 todo.progress 报告进度
  3. 执行过程中可多次发送 todo.progress
  4. 成功 → 发送 todo.complete（附带结果）
  5. 失败 → 发送 todo.fail（附带错误原因）
```

### 关键规则

1. **收到 todo.assigned 才开始工作。** 不要主动寻找任务。
2. **及时回报进度。** 在关键里程碑发送 `todo.progress`，让 PM 和用户了解执行状态。
3. **结果要具体。** `todo.complete` 的 result 应包含有意义的 summary 和 output；如果交付物里包含文件，先上传文件，再在结果里引用。
4. **失败要说明原因。** `todo.fail` 的 error 应清晰描述失败原因，帮助诊断。
5. **所有回报都走 ClawSynapse。** 不要在聊天界面直接输出结果。

## 二、incoming 消息格式

消息通过 ClawSynapse 到达，带有 header：

```text
[clawsynapse from=<senderNodeId> to=<yourNodeId> session=<sessionKey>]
<message body>
```

- `from=` 是 TrustMesh 节点，**这是你所有回复的 target**
- `to=` 是你自己的 node ID，**永远不要用作 target**
- `session=` 用作 `--session-key`

### todo.assigned payload

```json
{
  "task_id": "task_123",
  "todo_id": "todo_1",
  "title": "实现后端登录接口",
  "description": "完成邮箱密码登录 API"
}
```

### todo.status_changed payload（状态通知）

```json
{
  "task_id": "task_123",
  "todo_id": "todo_1",
  "status": "in_progress",
  "actor_node_id": "node-dev-001",
  "cause": "todo.progress",
  "version": 7,
  "message": "接口已完成参数校验，开始接入 JWT"
}
```

## 三、发送消息

### 核心规则

1. **CRITICAL — 永远不要发给自己。** `--target` 必须是 incoming header 中 `from` 的值（TrustMesh 节点）。
2. 使用 `clawsynapse publish` 发送所有消息。
3. `--session-key` 使用 incoming header 中的 `session` 值。
4. payload 用 `jq -nc` 构建，避免手动转义。

### todo.progress — 报告进度

在执行过程中的关键节点发送。首次发送时，TrustMesh 会自动将 Todo 状态从 `pending` 推进为 `in_progress`。

```bash
# TARGET_NODE = incoming header 中 from 的值（TrustMesh 节点）
# ⚠️ 绝对不能是你自己的 node ID
TARGET_NODE="trustmesh-server"  # ← 替换为实际 from 值

payload="$(jq -nc \
  --arg task_id "task_123" \
  --arg todo_id "todo_1" \
  --arg message "接口已完成参数校验，开始接入 JWT" \
  '{
    task_id: $task_id,
    todo_id: $todo_id,
    message: $message
  }')"

clawsynapse publish \
  --target "$TARGET_NODE" \
  --type todo.progress \
  --session-key task_123 \
  --message "$payload"
```

### todo.complete — 报告完成

Todo 执行成功时发送。

```bash
TARGET_NODE="trustmesh-server"  # ← 替换为实际 from 值

result='{
  "summary": "登录接口已完成",
  "output": "实现了注册、登录、JWT 校验",
  "artifact_refs": [
    {
      "artifact_id": "artifact_login_api",
      "kind": "report",
      "label": "登录接口实现说明"
    }
  ],
  "metadata": {
    "model": "claude-opus-4-6",
    "duration_ms": 1200
  }
}'

payload="$(jq -nc \
  --arg task_id "task_123" \
  --arg todo_id "todo_1" \
  --argjson result "$result" \
  '{
    task_id: $task_id,
    todo_id: $todo_id,
    result: $result
  }')"

clawsynapse publish \
  --target "$TARGET_NODE" \
  --type todo.complete \
  --session-key task_123 \
  --message "$payload"
```

result 字段说明：
- `summary`：一句话总结完成了什么
- `output`：详细描述执行结果
- `artifact_refs`：交付物引用列表（可选）
  - `artifact_id`：交付物唯一标识
  - `kind`：类型（`report`, `code`, `config` 等）
  - `label`：人类可读的标签
- `metadata`：执行元数据（可选），如使用的模型、耗时等

### 文件交付（可选）

如果 Todo 的交付结果包含本地文件（如报告、补丁包、截图、日志、导出数据），并且需要把文件实际交付给 TrustMesh 节点，执行顺序应为：

1. 先用 `clawsynapse transfer send` 把文件传给 incoming header 中 `from` 指定的 TrustMesh 节点
2. 再发送 `todo.complete`
3. 在 `result.artifact_refs` 中引用已上传文件，在 `result.metadata.transfers` 中附上结构化传输信息

规则：

- `transfer send` 的 `--target` 必须与 `publish` 一样，使用 incoming header 中 `from` 的值
- `artifact_refs[].kind` 对文件交付使用 `file`
- `artifact_refs[].artifact_id` 推荐直接使用 `transfer send` 返回的 `transferId`
- 不要新增 `todo.complete` 顶层字段；文件传输细节放进 `result.metadata`
- 如果文件是任务完成的必要交付，而上传失败，应发送 `todo.fail`
- 如果文件上传不是硬性要求但正文结果已足够完成任务，可继续 `todo.complete`，并在 `output` 或 `metadata` 里明确说明未上传原因

示例：先上传文件，再回报完成

```bash
TARGET_NODE="trustmesh-server"  # ← 替换为实际 from 值

transfer_json="$(clawsynapse --json transfer send \
  --target "$TARGET_NODE" \
  --file /tmp/login-api-report.pdf \
  --mime-type application/pdf)"

transfer_id="$(printf '%s' "$transfer_json" | jq -r '.data.transferId')"
transfer_bucket="$(printf '%s' "$transfer_json" | jq -r '.data.bucket')"
transfer_size="$(printf '%s' "$transfer_json" | jq -r '.data.size')"
transfer_checksum="$(printf '%s' "$transfer_json" | jq -r '.data.checksum')"

result="$(jq -nc \
  --arg summary "登录接口已完成" \
  --arg output "实现了注册、登录、JWT 校验，并已上传交付报告 PDF" \
  --arg transfer_id "$transfer_id" \
  --arg transfer_bucket "$transfer_bucket" \
  --argjson transfer_size "$transfer_size" \
  --arg transfer_checksum "$transfer_checksum" \
  '{
    summary: $summary,
    output: $output,
    artifact_refs: [
      {
        artifact_id: $transfer_id,
        kind: "file",
        label: "登录接口实现报告 PDF"
      }
    ],
    metadata: {
      transfers: [
        {
          transfer_id: $transfer_id,
          bucket: $transfer_bucket,
          size: $transfer_size,
          checksum: $transfer_checksum,
          purpose: "todo_deliverable"
        }
      ]
    }
  }')"

payload="$(jq -nc \
  --arg task_id "task_123" \
  --arg todo_id "todo_1" \
  --argjson result "$result" \
  '{
    task_id: $task_id,
    todo_id: $todo_id,
    result: $result
  }')"

clawsynapse publish \
  --target "$TARGET_NODE" \
  --type todo.complete \
  --session-key task_123 \
  --message "$payload"
```

多个文件时：

- 对每个文件分别执行一次 `clawsynapse --json transfer send`
- 把每个返回的 `transferId` 都写入 `artifact_refs`
- 需要排查时可用 `clawsynapse transfer get --id <transferId>` 或 `clawsynapse transfers`

### todo.fail — 报告失败

Todo 执行失败时发送。

```bash
TARGET_NODE="trustmesh-server"  # ← 替换为实际 from 值

payload="$(jq -nc \
  --arg task_id "task_123" \
  --arg todo_id "todo_1" \
  --arg error "Google OAuth 凭证缺失，无法完成 OAuth 接入" \
  '{
    task_id: $task_id,
    todo_id: $todo_id,
    error: $error
  }')"

clawsynapse publish \
  --target "$TARGET_NODE" \
  --type todo.fail \
  --session-key task_123 \
  --message "$payload"
```

### 获取结构化发送结果

```bash
clawsynapse --json publish \
  --target "$TARGET_NODE" \
  --type todo.progress \
  --session-key task_123 \
  --message "$payload"
```

## 四、Guardrails

- **永远不要用你自己的 node ID 作为 `--target`。** target 是 TrustMesh 节点（incoming `from`）。
- **不要发送 `conversation.reply`、`task.create`。** 这些是 PM Agent 的消息类型，不是执行 Agent 的。
- **Todo 终态不可逆。** 已经 `done` 或 `failed` 的 Todo 不能再更新，服务端会拒绝（`TODO_ALREADY_DONE` / `TODO_ALREADY_FAILED`）。
- 不要在聊天界面直接回复，必须使用 `clawsynapse publish`。
- 不要丢弃 `--session-key`。
- 不要发送协议中未定义的字段。
- payload 较大时用 `jq -nc` 构建。

## 五、常见错误码

| 错误码 | 含义 |
|--------|------|
| `BAD_PAYLOAD` | JSON 结构或必填字段无效 |
| `VALIDATION_ERROR` | 业务参数校验失败 |
| `FORBIDDEN` | 无权执行该操作（可能不是该 Todo 的指派 Agent） |
| `NOT_FOUND` | 目标 Task 或 Todo 不存在 |
| `TODO_FINALIZED` | Todo 已结束，不再接受更新 |
| `TODO_ALREADY_DONE` | Todo 已完成 |
| `TODO_ALREADY_FAILED` | Todo 已失败 |

## 六、Peer 发现

如果需要确认 TrustMesh 节点信息：

```bash
clawsynapse --json peers
```

## 七、重要提示

- 不要运行 `clawsynapsed`，它由系统管理。
- 你只能发送 `todo.progress`、`todo.complete`、`todo.fail` 三种消息类型。
- 任务规划和对话回复由 PM Agent 负责，不需要你介入。
