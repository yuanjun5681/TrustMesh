---
name: trustmesh
description: >
  Use this skill when an agent needs to send or reply to protocol messages over
  ClawSynapse. Covers how to use `clawsynapse publish`, how to preserve
  `sessionKey` across replies, and how to build protocol message bodies.
compatibility: Requires clawsynapse CLI and a running clawsynapsed daemon
metadata:
  author: TrustMesh
  version: "1.0"
allowed-tools:
  - "Bash(clawsynapse:*)"
---

# TrustMesh Messaging Over ClawSynapse

This skill is about one thing: how an agent should use `clawsynapse` to send or reply to protocol messages.

It does not assume who the remote node is. If you receive a ClawSynapse message, reply to the sender shown in the message header. If you need to proactively send a message, use the target node given by the current task or user context.

## When to Use This Skill

Use this skill whenever:
- You receive a ClawSynapse message and need to reply correctly
- You need to send a protocol message with `clawsynapse publish`
- You need to preserve a `sessionKey` across follow-up messages
- You need an exact JSON shape for one of the protocol message bodies

## Incoming ClawSynapse Message Format

Messages arrive with a header like this:

```text
[clawsynapse from=<senderNodeId> to=<localNodeId> session=<sessionKey>]
<message body>
```

Example:

```text
[clawsynapse from=node-2 to=node-1 session=conv_123]
{"conversation_id":"conv_123","content":"请先确认需求边界"}
```

Rules:
- `from=` is the node you reply to
- `session=` should be reused as `--session-key` in your reply
- Do not reply with plain text in the chat interface; use `clawsynapse publish`

## First Step: Resolve the Target

If the target node is not already known, inspect available peers:

```bash
clawsynapse --json peers
```

If you still cannot determine the right target node from current context, ask the user to clarify.

## Core Rules

1. **CRITICAL — Never send to yourself.** When replying to an incoming message, `--target` MUST be the `from` value from the incoming `[clawsynapse from=<senderNodeId> ...]` header. This applies to ALL message types (`conversation.reply`, `todo.progress`, `todo.complete`, `todo.fail`, `task.create`, etc.). Your own node ID appears in the `to` field — never use that as the target.
2. Use `clawsynapse publish` for every outbound protocol message.
3. Keep the business payload in `--message` as a valid JSON string.
4. If the incoming message has `session=...`, always include `--session-key` with the same value.
5. For a new thread without an incoming session, prefer a stable business ID such as `conversation_id` or `task_id` as the `--session-key`.
6. Keep the payload exact. Do not rename fields or add undocumented ones.
7. Use `clawsynapse --json publish` when you need the returned `messageId`.

## Basic Command Patterns

### Reply to an incoming message

Extract `from` from the incoming header and use it as `TARGET_NODE`.

**Example:** if the incoming header is `[clawsynapse from=node-platform to=node-2 session=conv_123]`, then `TARGET_NODE="node-platform"`. Do NOT use your own node ID (`node-2` in this example).

```bash
# TARGET_NODE = the "from" value in the incoming [clawsynapse from=... ] header
# ⚠️ This must NOT be your own node ID — that would send the message to yourself
TARGET_NODE="node-platform"  # ← replace with actual "from" value from incoming header

payload="$(jq -nc --arg conversation_id "conv_123" --arg content "我先确认两个边界问题。" '{
  conversation_id: $conversation_id,
  content: $content
}')"

clawsynapse publish \
  --target "$TARGET_NODE" \
  --type conversation.reply \
  --session-key conv_123 \
  --message "$payload"
```

### Start a new protocol message thread

```bash
# TARGET = the node you want to reach (from peers list or task context)
TARGET_NODE="trustmesh"  # ← replace with actual target node from task context or peers

payload="$(jq -nc --arg task_id "task_123" --arg todo_id "todo_1" --arg message "开始执行" '{
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

### Get machine-readable publish output

```bash
clawsynapse --json publish \
  --target "$TARGET_NODE" \
  --type todo.progress \
  --session-key task_123 \
  --message "$payload"
```

## Protocol Message Types

Use the following `--type` values when publishing:

| type | When to use it |
|------|----------------|
| `conversation.reply` | Send a conversation reply |
| `task.create` | Create a task with todos |
| `todo.progress` | Report progress for a todo |
| `todo.complete` | Report successful completion of a todo |
| `todo.fail` | Report failure of a todo |

You may also receive these message types:

| type | Meaning |
|------|---------|
| `conversation.message` | An incoming conversation message |
| `task.created` | Confirmation that a task was created |
| `task.updated` | Task status changed |
| `todo.assigned` | A todo was assigned |
| `todo.updated` | A todo status changed |

## Payload Formats

The examples below are the JSON bodies that go inside `--message`.

### `conversation.reply`

```json
{
  "conversation_id": "conv_123",
  "content": "我已理解当前需求，先确认两个边界问题。"
}
```

```bash
payload="$(jq -nc --arg conversation_id "conv_123" --arg content "我已理解当前需求，先确认两个边界问题。" '{
  conversation_id: $conversation_id,
  content: $content
}')"

clawsynapse publish --target "$TARGET_NODE" --type conversation.reply --session-key conv_123 --message "$payload"
```

### `task.create`

```json
{
  "project_id": "proj_123",
  "conversation_id": "conv_123",
  "title": "实现用户登录",
  "description": "支持邮箱密码和 Google OAuth",
  "todos": [
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
  ]
}
```

```bash
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

clawsynapse publish --target "$TARGET_NODE" --type task.create --session-key conv_123 --message "$payload"
```

### `todo.progress`

```json
{
  "task_id": "task_123",
  "todo_id": "todo_1",
  "message": "接口已完成参数校验，开始接入 JWT"
}
```

```bash
payload="$(jq -nc \
  --arg task_id "task_123" \
  --arg todo_id "todo_1" \
  --arg message "接口已完成参数校验，开始接入 JWT" \
  '{
    task_id: $task_id,
    todo_id: $todo_id,
    message: $message
  }')"

clawsynapse publish --target "$TARGET_NODE" --type todo.progress --session-key task_123 --message "$payload"
```

### `todo.complete`

```json
{
  "task_id": "task_123",
  "todo_id": "todo_1",
  "result": {
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
      "model": "gpt-5",
      "duration_ms": 1200
    }
  }
}
```

```bash
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
    "model": "gpt-5",
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

clawsynapse publish --target "$TARGET_NODE" --type todo.complete --session-key task_123 --message "$payload"
```

### `todo.fail`

```json
{
  "task_id": "task_123",
  "todo_id": "todo_1",
  "error": "Google OAuth 凭证缺失"
}
```

```bash
payload="$(jq -nc \
  --arg task_id "task_123" \
  --arg todo_id "todo_1" \
  --arg error "Google OAuth 凭证缺失" \
  '{
    task_id: $task_id,
    todo_id: $todo_id,
    error: $error
  }')"

clawsynapse publish --target "$TARGET_NODE" --type todo.fail --session-key task_123 --message "$payload"
```

## Common Incoming Payloads

These are examples of message bodies you may receive.

### `conversation.message`

```json
{
  "conversation_id": "conv_123",
  "project_id": "proj_123",
  "content": "你正在处理首次需求澄清。",
  "user_content": "我需要一个用户登录功能",
  "is_initial_message": true
}
```

### `todo.assigned`

```json
{
  "task_id": "task_123",
  "todo_id": "todo_1",
  "title": "实现后端登录接口",
  "description": "完成邮箱密码登录 API"
}
```

### `task.updated`

```json
{
  "task_id": "task_123",
  "status": "in_progress"
}
```

### `todo.updated`

```json
{
  "task_id": "task_123",
  "todo_id": "todo_1",
  "status": "in_progress",
  "message": "接口已完成参数校验，开始接入 JWT"
}
```

## Reply Workflow

1. Read the incoming ClawSynapse header: `[clawsynapse from=<sender> to=<you> session=<key>]`
2. Set `TARGET_NODE` to the `from` value (the sender). **Never use the `to` value — that is you.**
3. Set `--session-key` to the `session` value.
4. Build the correct JSON payload for the business message.
5. Publish with `--target "$TARGET_NODE"` and the matching `--type`.

## Guardrails

- **Never use your own node ID as `--target`.** The target is always the remote node (the `from` field of the incoming message). Sending to yourself is the most common mistake — double-check before publishing.
- Do not reply in plain chat text when the remote side expects a ClawSynapse message.
- Do not drop `--session-key` on follow-up replies.
- Do not send malformed JSON in `--message`.
- Do not send fields that are not in the protocol.
- If the payload is large or nested, build it with `jq -nc` instead of manual string escaping.

## Common Errors

| Error code | Meaning |
|-----------|---------|
| `BAD_PAYLOAD` | The JSON structure or required fields are invalid |
| `VALIDATION_ERROR` | Business validation failed |
| `FORBIDDEN` | The sender is not allowed to perform that action |
| `NOT_FOUND` | The referenced conversation, task, or todo does not exist |
| `CONVERSATION_TASK_EXISTS` | A task already exists for that conversation |
| `CONVERSATION_RESOLVED` | The conversation is already closed |
| `TODO_FINALIZED` | The todo already reached a terminal state |
| `TODO_ALREADY_DONE` | The todo is already completed |
| `TODO_ALREADY_FAILED` | The todo is already failed |

## Important Notes

- Do not run `clawsynapsed`; it is managed separately.
- Use `clawsynapse --json peers` when you need to inspect available peers.
- Use `clawsynapse --json publish` when you need structured publish results.
