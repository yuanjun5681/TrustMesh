package model

import "time"

type User struct {
	ID           string    `json:"id" bson:"_id"`
	Email        string    `json:"email" bson:"email"`
	Name         string    `json:"name" bson:"name"`
	PasswordHash string    `json:"-" bson:"password_hash"`
	CreatedAt    time.Time `json:"created_at" bson:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" bson:"updated_at"`
}

type PMAgentSummary struct {
	ID     string `json:"id" bson:"id"`
	Name   string `json:"name" bson:"name"`
	NodeID string `json:"node_id" bson:"node_id"`
	Status string `json:"status" bson:"status"`
}

type Project struct {
	ID          string         `json:"id" bson:"_id"`
	UserID      string         `json:"-" bson:"user_id"`
	Name        string         `json:"name" bson:"name"`
	Description string         `json:"description" bson:"description"`
	Status      string         `json:"status" bson:"status"`
	PMAgentID   string         `json:"-" bson:"pm_agent_id"`
	PMAgent     PMAgentSummary `json:"pm_agent" bson:"pm_agent"`
	CreatedAt   time.Time      `json:"created_at" bson:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at" bson:"updated_at"`
}

type ConversationMessage struct {
	ID        string    `json:"id" bson:"id"`
	Role      string    `json:"role" bson:"role"`
	Content   string    `json:"content" bson:"content"`
	CreatedAt time.Time `json:"created_at" bson:"created_at"`
}

type TaskSummary struct {
	ID                 string    `json:"id" bson:"id"`
	Title              string    `json:"title" bson:"title"`
	Status             string    `json:"status" bson:"status"`
	Priority           string    `json:"priority" bson:"priority"`
	TodoCount          int       `json:"todo_count" bson:"todo_count"`
	CompletedTodoCount int       `json:"completed_todo_count" bson:"completed_todo_count"`
	CreatedAt          time.Time `json:"created_at" bson:"created_at"`
	UpdatedAt          time.Time `json:"updated_at" bson:"updated_at"`
}

type ConversationListItem struct {
	ID          string              `json:"id" bson:"id"`
	ProjectID   string              `json:"project_id" bson:"project_id"`
	Status      string              `json:"status" bson:"status"`
	LastMessage ConversationMessage `json:"last_message" bson:"last_message"`
	LinkedTask  *TaskSummary        `json:"linked_task" bson:"linked_task"`
	CreatedAt   time.Time           `json:"created_at" bson:"created_at"`
	UpdatedAt   time.Time           `json:"updated_at" bson:"updated_at"`
}

type ConversationDetail struct {
	ID         string                `json:"id" bson:"id"`
	ProjectID  string                `json:"project_id" bson:"project_id"`
	Status     string                `json:"status" bson:"status"`
	Messages   []ConversationMessage `json:"messages" bson:"messages"`
	LinkedTask *TaskSummary          `json:"linked_task" bson:"linked_task"`
	CreatedAt  time.Time             `json:"created_at" bson:"created_at"`
	UpdatedAt  time.Time             `json:"updated_at" bson:"updated_at"`
}

type ConversationStreamSnapshot struct {
	Conversation ConversationDetail `json:"conversation"`
}

type Agent struct {
	ID           string     `json:"id" bson:"_id"`
	UserID       string     `json:"-" bson:"user_id"`
	Name         string     `json:"name" bson:"name"`
	Description  string     `json:"description" bson:"description"`
	Role         string     `json:"role" bson:"role"`
	Capabilities []string   `json:"capabilities" bson:"capabilities"`
	NodeID       string     `json:"node_id" bson:"node_id"`
	Status       string     `json:"status" bson:"status"`
	LastSeenAt   *time.Time `json:"last_seen_at" bson:"last_seen_at"`
	Usage        AgentUsage `json:"usage" bson:"-"`
	CreatedAt    time.Time  `json:"created_at" bson:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at" bson:"updated_at"`
}

type AgentUsage struct {
	ProjectCount int  `json:"project_count" bson:"-"`
	TaskCount    int  `json:"task_count" bson:"-"`
	TodoCount    int  `json:"todo_count" bson:"-"`
	TotalCount   int  `json:"total_count" bson:"-"`
	InUse        bool `json:"in_use" bson:"-"`
}

type TodoAssignee struct {
	AgentID string `json:"agent_id" bson:"agent_id"`
	Name    string `json:"name" bson:"name"`
	NodeID  string `json:"node_id" bson:"node_id"`
}

type TodoResultArtifactRef struct {
	ArtifactID string `json:"artifact_id" bson:"artifact_id"`
	Kind       string `json:"kind" bson:"kind"`
	Label      string `json:"label" bson:"label"`
}

type TodoResult struct {
	Summary      string                  `json:"summary" bson:"summary"`
	Output       string                  `json:"output" bson:"output"`
	ArtifactRefs []TodoResultArtifactRef `json:"artifact_refs" bson:"artifact_refs"`
	Metadata     map[string]any          `json:"metadata" bson:"metadata"`
}

type Todo struct {
	ID          string       `json:"id" bson:"id"`
	Title       string       `json:"title" bson:"title"`
	Description string       `json:"description" bson:"description"`
	Status      string       `json:"status" bson:"status"`
	Assignee    TodoAssignee `json:"assignee" bson:"assignee"`
	StartedAt   *time.Time   `json:"started_at" bson:"started_at"`
	CompletedAt *time.Time   `json:"completed_at" bson:"completed_at"`
	FailedAt    *time.Time   `json:"failed_at" bson:"failed_at"`
	Error       *string      `json:"error" bson:"error"`
	Result      TodoResult   `json:"result" bson:"result"`
	CreatedAt   time.Time    `json:"created_at" bson:"created_at"`
}

type TaskArtifact struct {
	ID           string         `json:"id" bson:"id"`
	SourceTodoID *string        `json:"source_todo_id" bson:"source_todo_id"`
	Kind         string         `json:"kind" bson:"kind"`
	Title        string         `json:"title" bson:"title"`
	URI          string         `json:"uri" bson:"uri"`
	MimeType     *string        `json:"mime_type" bson:"mime_type"`
	Metadata     map[string]any `json:"metadata" bson:"metadata"`
}

type TaskResult struct {
	Summary     string         `json:"summary" bson:"summary"`
	FinalOutput string         `json:"final_output" bson:"final_output"`
	Metadata    map[string]any `json:"metadata" bson:"metadata"`
}

type TaskListItem struct {
	ID                 string         `json:"id" bson:"id"`
	ProjectID          string         `json:"project_id" bson:"project_id"`
	ConversationID     string         `json:"conversation_id" bson:"conversation_id"`
	Title              string         `json:"title" bson:"title"`
	Description        string         `json:"description" bson:"description"`
	Status             string         `json:"status" bson:"status"`
	Priority           string         `json:"priority" bson:"priority"`
	PMAgent            PMAgentSummary `json:"pm_agent" bson:"pm_agent"`
	TodoCount          int            `json:"todo_count" bson:"todo_count"`
	CompletedTodoCount int            `json:"completed_todo_count" bson:"completed_todo_count"`
	FailedTodoCount    int            `json:"failed_todo_count" bson:"failed_todo_count"`
	CreatedAt          time.Time      `json:"created_at" bson:"created_at"`
	UpdatedAt          time.Time      `json:"updated_at" bson:"updated_at"`
}

type TaskDetail struct {
	ID             string         `json:"id" bson:"_id"`
	UserID         string         `json:"-" bson:"user_id"`
	ProjectID      string         `json:"project_id" bson:"project_id"`
	ConversationID string         `json:"conversation_id" bson:"conversation_id"`
	Title          string         `json:"title" bson:"title"`
	Description    string         `json:"description" bson:"description"`
	Status         string         `json:"status" bson:"status"`
	Priority       string         `json:"priority" bson:"priority"`
	PMAgentID      string         `json:"-" bson:"pm_agent_id"`
	PMAgent        PMAgentSummary `json:"pm_agent" bson:"pm_agent"`
	Todos          []Todo         `json:"todos" bson:"todos"`
	Artifacts      []TaskArtifact `json:"artifacts" bson:"artifacts"`
	Result         TaskResult     `json:"result" bson:"result"`
	Version        int            `json:"version" bson:"version"`
	CreatedAt      time.Time      `json:"created_at" bson:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at" bson:"updated_at"`
}

type TaskEvent struct {
	ID        string         `json:"id" bson:"_id"`
	TaskID    string         `json:"task_id" bson:"task_id"`
	ActorType string         `json:"actor_type" bson:"actor_type"`
	ActorID   string         `json:"actor_id" bson:"actor_id"`
	EventType string         `json:"event_type" bson:"event_type"`
	Content   *string        `json:"content" bson:"content"`
	Metadata  map[string]any `json:"metadata" bson:"metadata"`
	CreatedAt time.Time      `json:"created_at" bson:"created_at"`
}

type TaskStreamSnapshot struct {
	Task   TaskDetail  `json:"task"`
	Events []TaskEvent `json:"events"`
}

// Internal conversation record for mutable state.
type Conversation struct {
	ID        string                `bson:"_id"`
	UserID    string                `bson:"user_id"`
	ProjectID string                `bson:"project_id"`
	Status    string                `bson:"status"`
	Messages  []ConversationMessage `bson:"messages"`
	CreatedAt time.Time             `bson:"created_at"`
	UpdatedAt time.Time             `bson:"updated_at"`
}
