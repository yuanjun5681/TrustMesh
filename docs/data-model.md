# 数据模型

## 实体关系图

```
User 1───N Project                (用户创建项目)
Project 1───1 Agent(PM)           (每个项目绑定一个项目经理 Agent)
Project 1───N Task                (项目包含任务)
Task ◇───N TodoItem               (任务内嵌 Todo 列表)
Task 1───N TaskEvent              (任务活动流)
Agent 1───N AgentSession          (Agent 执行会话，可选)
AgentSession N───1 Task           (会话关联任务)
Project 1───N Conversation        (用户与项目经理的对话)
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
  "type": "string",                    // "llm" | "script" | "webhook"
  "config": {
    "model": "string",
    "system_prompt": "string",
    "tools": ["string"],
    "temperature": 0.7
  },
  "node_id": "string",                // Agent 所在节点 ID，平台内唯一
  "status": "string",                 // "online" | "offline" | "busy"
  "last_seen_at": "datetime | null",
  "heartbeat_at": "datetime | null",
  "owner_id": "ObjectId ref:users",
  "created_at": "datetime",
  "updated_at": "datetime"
}
```

约束：
- `node_id` 全局唯一，用来把 NATS 主题中的节点身份映射到平台内 Agent 记录。
- Agent 通过心跳维持在线状态；服务端根据最近一次 `heartbeat_at` 推导 `status`。
- `role=pm` 表示该 Agent 可作为项目经理 Agent，被项目绑定后承担需求规划职责。
- 用户创建 Agent 时必须提供 `node_id`。
- 用户可编辑字段：`name`、`role`、`description`、`capabilities`、`config`。
- 非系统迁移场景下不允许直接修改 `node_id`，避免已存在任务与 NATS 身份映射失效。

### projects

```json
{
  "_id": "ObjectId",
  "name": "string",
  "description": "string",
  "status": "string",                 // "active" | "archived"
  "owner_id": "ObjectId ref:users",
  "pm_agent_id": "ObjectId ref:agents",
  "pm_agent_node_id": "string",
  "created_at": "datetime",
  "updated_at": "datetime"
}
```

约束：
- `pm_agent_id` 对应的 Agent 必须满足 `role=pm`。
- 只有当项目 PM Agent 当前在线时，才允许创建 Conversation 或发送用户需求消息。

### tasks（核心模型）

MVP 中所有任务都由 PM Agent 创建，并且包含 Todo 列表。即使是简单任务，也用单个 Todo 表示，统一执行模型。

```json
{
  "_id": "ObjectId",
  "project_id": "ObjectId ref:projects",
  "conversation_id": "ObjectId ref:conversations",
  "title": "string",
  "description": "string (Markdown)",
  "status": "string",                 // "pending" | "in_progress" | "done" | "failed"
  "priority": "string",               // "low" | "medium" | "high" | "urgent"
  "created_by_type": "string",        // "user" | "pm_agent"
  "created_by_id": "ObjectId",
  "pm_agent_id": "ObjectId ref:agents",
  "due_date": "datetime | null",

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
        "artifacts": [],
        "metadata": {}
      },
      "created_at": "datetime",
      "completed_at": "datetime | null",
      "failed_at": "datetime | null",
      "error": "string | null"
    }
  ],

  "result": {
    "summary": "string",
    "todo_results": [
      {
        "todo_id": "string",
        "agent_id": "ObjectId",
        "status": "string",
        "summary": "string",
        "output": "string"
      }
    ]
  },

  "version": 1,
  "created_at": "datetime",
  "updated_at": "datetime"
}
```

结果落库约定：
- 执行 Agent 通过 `todo.complete` 上报的完整 `result`，原样保存到 `todos[].result`。
- `tasks.result` 只保存任务级聚合结果，作为看板和详情页的摘要视图，不重复保存完整 artifacts。
- 如需展示完整单个 Todo 成果，应优先读取对应 `todos[].result`。

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
                                       // | "todo_failed" | "conversation_message"
  "payload": {},
  "created_at": "datetime"
}
```

### agent_sessions

```json
{
  "_id": "ObjectId",
  "agent_id": "ObjectId ref:agents",
  "task_id": "ObjectId ref:tasks",
  "todo_id": "string",
  "status": "string",                 // "running" | "completed" | "failed" | "cancelled"
  "started_at": "datetime",
  "finished_at": "datetime | null",
  "logs": [{}],
  "error": "string | null"
}
```

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
db.tasks.createIndex({ conversation_id: 1 })
db.tasks.createIndex({ pm_agent_id: 1 })
db.tasks.createIndex({ "todos.assignee_agent_id": 1, "todos.status": 1 })
db.tasks.createIndex({ status: 1 })

// task_events
db.task_events.createIndex({ task_id: 1, created_at: 1 })

// agent_sessions
db.agent_sessions.createIndex({ agent_id: 1 })
db.agent_sessions.createIndex({ task_id: 1 })

// conversations
db.conversations.createIndex({ project_id: 1, user_id: 1 })
```
