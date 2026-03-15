# 数据模型

## 实体关系图

```
User 1───N Project                (用户创建项目)
Project 1───1 Agent(PM)           (每个项目绑定一个项目经理 Agent)
Project 1───N Task                (项目包含任务)
Task ◇───N TodoItem               (复杂任务内嵌 Todo 列表)
Task N───1 Agent                  (任务指派给 Agent)
Task N───1 User                   (任务由人创建，也可指派给人)
Task 1───N TaskEvent              (任务活动流)
Agent 1───N AgentSession          (Agent 执行会话)
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
  "type": "string",                    // "llm" | "script" | "webhook"
  "config": {
    "model": "string",
    "system_prompt": "string",
    "tools": ["string"],
    "temperature": 0.7
  },
  "api_key_hash": "string",
  "node_id": "string",                // ClawSynapse 节点 ID
  "status": "string",                 // "online" | "offline" | "busy"
  "owner_id": "ObjectId ref:users",
  "created_at": "datetime",
  "updated_at": "datetime"
}
```

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

### tasks（核心模型）

任务分两种模式：
- **简单任务**：直接指派给 Agent 执行，`todos` 为空数组
- **复杂任务**：由 PM Agent 拆分为 Todo 列表，每个 Todo 可独立指派

```json
{
  "_id": "ObjectId",
  "project_id": "ObjectId ref:projects",
  "title": "string",
  "description": "string (Markdown)",
  "status": "string",                 // "todo" | "in_progress" | "done" | "failed"
  "priority": "string",               // "low" | "medium" | "high" | "urgent"
  "assignee_type": "string | null",   // "user" | "agent"
  "assignee_id": "ObjectId | null",
  "created_by": "ObjectId ref:users",
  "due_date": "datetime | null",

  "todos": [
    {
      "id": "string (uuid)",
      "title": "string",
      "description": "string",
      "status": "string",             // "pending" | "in_progress" | "done" | "skipped"
      "assignee_type": "string",
      "assignee_id": "ObjectId",
      "result": {},
      "created_at": "datetime",
      "completed_at": "datetime | null"
    }
  ],

  "result": {
    "summary": "string",
    "output": "string",
    "artifacts": [
      { "type": "string", "name": "string", "content": "string" }
    ],
    "metadata": {
      "model": "string",
      "tokens_used": 0,
      "duration_ms": 0
    }
  },

  "version": 1,
  "created_at": "datetime",
  "updated_at": "datetime"
}
```

### task_events

```json
{
  "_id": "ObjectId",
  "task_id": "ObjectId ref:tasks",
  "actor_type": "string",             // "user" | "agent" | "system"
  "actor_id": "ObjectId",
  "event_type": "string",             // "status_change" | "comment" | "assignment"
                                       // | "progress_update" | "result_submitted"
                                       // | "todo_updated"
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
db.agents.createIndex({ owner_id: 1 })
db.agents.createIndex({ status: 1 })
db.agents.createIndex({ role: 1 })

// projects
db.projects.createIndex({ owner_id: 1 })
db.projects.createIndex({ pm_agent_id: 1 })

// tasks
db.tasks.createIndex({ project_id: 1, status: 1 })
db.tasks.createIndex({ assignee_type: 1, assignee_id: 1 })
db.tasks.createIndex({ "todos.assignee_type": 1, "todos.assignee_id": 1 })
db.tasks.createIndex({ status: 1 })

// task_events
db.task_events.createIndex({ task_id: 1, created_at: 1 })

// agent_sessions
db.agent_sessions.createIndex({ agent_id: 1 })
db.agent_sessions.createIndex({ task_id: 1 })

// conversations
db.conversations.createIndex({ project_id: 1, user_id: 1 })
```
