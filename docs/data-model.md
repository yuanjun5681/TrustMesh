# 数据模型

## 实体关系图

```
User 1───N Project                (用户创建项目)
Project 1───1 Agent(PM)           (每个项目绑定一个项目经理 Agent)
Project 1───N Conversation        (用户与项目经理的对话)
Conversation 1───1 Task           (一次需求对话最终沉淀为一个任务)
Project 1───N Task                (项目包含任务)
Task ◇───N TodoItem               (任务内嵌 Todo 列表)
Task 1───N TaskEvent              (任务活动流)
```

## 文档模型定义（MongoDB）

### users

```json
{
  "_id": "ObjectId",
  "email": "string (unique)",
  "name": "string",
  "password_hash": "string",
  "created_at": "datetime",
  "updated_at": "datetime"
}
```

### agents

```json
{
  "_id": "ObjectId",
  "name": "string",
  "description": "string",
  "role": "string",                    // "pm" | "developer" | "reviewer" | "custom"
  "capabilities": ["string"],          // 例如 "backend", "frontend", "qa", "oauth"
  "node_id": "string",                // Agent 所在节点 ID，平台内唯一
  "status": "string",                 // "online" | "offline" | "busy"
  "last_seen_at": "datetime | null",
  "owner_id": "ObjectId ref:users",
  "created_at": "datetime",
  "updated_at": "datetime"
}
```

约束：
- `node_id` 全局唯一，对应该 Agent 的 ClawSynapse nodeId，用于消息路由和身份映射。
- Agent 在线状态由 ClawSynapse 的节点发现机制（`discovery.announce`）维护；TrustMesh 后端通过 `GET /v1/peers` 定期同步，更新 `status` 和 `last_seen_at`。
- `role=pm` 表示该 Agent 可作为项目经理 Agent，被项目绑定后承担需求规划职责。
- 用户创建 Agent 时必须提供 `node_id`。
- 用户可编辑字段：`name`、`role`、`description`、`capabilities`。
- 非系统迁移场景下不允许直接修改 `node_id`，避免已存在任务与 ClawSynapse 身份映射失效。

### projects

```json
{
  "_id": "ObjectId",
  "name": "string",
  "description": "string",
  "status": "string",                 // "active" | "archived"
  "owner_id": "ObjectId ref:users",
  "pm_agent_id": "ObjectId ref:agents",
  "created_at": "datetime",
  "updated_at": "datetime"
}
```

约束：
- `pm_agent_id` 对应的 Agent 必须满足 `role=pm`。
- 只有当项目 PM Agent 当前在线时（在 ClawSynapse peers 列表中可见），才允许创建 Conversation 或发送用户需求消息。

### tasks（核心模型）

MVP 中所有任务都由 PM Agent 创建，并且包含 Todo 列表。即使是简单任务，也用单个 Todo 表示，统一执行模型。业务上一次 `Conversation` 最终只对应一个 `Task`。

```json
{
  "_id": "ObjectId",
  "project_id": "ObjectId ref:projects",
  "conversation_id": "ObjectId ref:conversations",
  "title": "string",
  "description": "string (Markdown)",
  "status": "string",                 // "pending" | "in_progress" | "done" | "failed"
  "priority": "string",               // "low" | "medium" | "high" | "urgent"
  "pm_agent_id": "ObjectId ref:agents",

  "todos": [
    {
      "id": "string (uuid)",
      "title": "string",
      "description": "string",
      "status": "string",             // "pending" | "in_progress" | "done" | "failed"
      "assignee_agent_id": "ObjectId ref:agents",
      "assignee_node_id": "string",
      "started_at": "datetime | null",
      "result": {
        "summary": "string",
        "output": "string",
        "artifact_refs": [
          {
            "artifact_id": "string",
            "kind": "string",         // "file" | "link" | "log" | "report"
            "label": "string"
          }
        ],
        "metadata": {}
      },
      "created_at": "datetime",
      "completed_at": "datetime | null",
      "failed_at": "datetime | null",
      "error": "string | null"
    }
  ],

  "artifacts": [
    {
      "id": "string",
      "source_todo_id": "string | null",
      "kind": "string",               // "file" | "link" | "log" | "report"
      "title": "string",
      "uri": "string",
      "mime_type": "string | null",
      "metadata": {}
    }
  ],

  "result": {
    "summary": "string",
    "final_output": "string",
    "metadata": {}
  },

  "version": 1,
  "created_at": "datetime",
  "updated_at": "datetime"
}
```

约束：
- `conversation_id` 必填，且一个 `Conversation` 最多只能生成一个 `Task`。
- 一旦 Task 创建完成，`Conversation` 与 `Task` 关系视为 1:1。
- `project_id` 必须与 `conversation_id` 对应 Conversation 的 `project_id` 一致。
- `pm_agent_id` 表示创建该 Task 的 PM Agent，且必须与对应 Project 绑定的 PM Agent 一致。

结果落库约定：
- 执行 Agent 通过 `todo.complete` 上报的执行结果，保存到 `todos[].result`，作为 Todo 级执行真相源。
- `todos[].result.artifact_refs` 只保存交付物引用，不保存大体积交付物内容。
- `tasks.artifacts` 统一管理任务最终交付物，可追溯到 `source_todo_id`。
- `tasks.result` 只保存任务级最终摘要与汇总结论，不重复保存每个 Todo 的详细结果。
- 用户侧优先查看 `tasks.result` 和 `tasks.artifacts`；排查执行细节时再读取对应 `todos[].result`。

### task_events

```json
{
  "_id": "ObjectId",
  "task_id": "ObjectId ref:tasks",
  "actor_type": "string",             // "user" | "agent" | "system"
  "actor_id": "ObjectId",
  "event_type": "string",             // "task_created" | "task_status_changed"
                                       // | "todo_assigned" | "todo_started"
                                       // | "todo_progress" | "todo_completed"
                                       // | "todo_failed"
                                       // | "task_comment"
  "content": "string | null",
  "metadata": {},
  "created_at": "datetime"
}
```

说明：
- Task 评论建议作为 `task_events` 中的 `task_comment` 事件保存。
- 评论中如需引用交付物，应引用 `tasks.artifacts[].id`，不重复保存附件实体。
- `content` 用于时间线展示的人类可读文案。
- `metadata` 用于结构化扩展字段，例如 `todo_id`、状态变更前后值、`artifact_ids` 等。

### conversations（用户与 PM Agent 对话）

```json
{
  "_id": "ObjectId",
  "project_id": "ObjectId ref:projects",
  "user_id": "ObjectId ref:users",
  "messages": [
    {
      "id": "string (uuid)",
      "role": "string",               // "user" | "pm_agent"
      "content": "string",
      "created_at": "datetime"
    }
  ],
  "status": "string",                 // "active" | "resolved"
  "created_at": "datetime",
  "updated_at": "datetime"
}
```

约束：
- 一个 `Conversation` 归属一个 `Project`。
- 一个 `Conversation` 在生命周期内最多只关联一个 `Task`；Task 创建前允许暂时不存在对应任务。

## 索引设计

```javascript
// users
db.users.createIndex({ email: 1 }, { unique: true })

// agents
db.agents.createIndex({ node_id: 1 }, { unique: true })
db.agents.createIndex({ owner_id: 1 })
db.agents.createIndex({ status: 1 })
db.agents.createIndex({ role: 1 })
db.agents.createIndex({ capabilities: 1 })

// projects
db.projects.createIndex({ owner_id: 1 })
db.projects.createIndex({ pm_agent_id: 1 })

// tasks
db.tasks.createIndex({ project_id: 1, status: 1 })
db.tasks.createIndex({ conversation_id: 1 }, { unique: true })
db.tasks.createIndex({ pm_agent_id: 1 })
db.tasks.createIndex({ "todos.assignee_agent_id": 1, "todos.status": 1 })
db.tasks.createIndex({ status: 1 })

// task_events
db.task_events.createIndex({ task_id: 1, created_at: 1 })

// conversations
db.conversations.createIndex({ project_id: 1, user_id: 1 })
```
