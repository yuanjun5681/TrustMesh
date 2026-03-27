package model

import "time"

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
	ID           string       `json:"id" bson:"id"`
	Order        int          `json:"order" bson:"order"`
	Title        string       `json:"title" bson:"title"`
	Description  string       `json:"description" bson:"description"`
	Status       string       `json:"status" bson:"status"`
	Assignee     TodoAssignee `json:"assignee" bson:"assignee"`
	StartedAt    *time.Time   `json:"started_at" bson:"started_at"`
	CompletedAt  *time.Time   `json:"completed_at" bson:"completed_at"`
	FailedAt     *time.Time   `json:"failed_at" bson:"failed_at"`
	CanceledAt   *time.Time   `json:"canceled_at" bson:"canceled_at"`
	Error        *string      `json:"error" bson:"error"`
	CancelReason *string      `json:"cancel_reason" bson:"cancel_reason"`
	Result       TodoResult   `json:"result" bson:"result"`
	CreatedAt    time.Time    `json:"created_at" bson:"created_at"`
}

type ActorRef struct {
	ActorType string `json:"actor_type" bson:"actor_type"`
	ActorID   string `json:"actor_id" bson:"actor_id"`
	ActorName string `json:"actor_name" bson:"actor_name"`
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
	ConversationID string         `json:"conversation_id,omitempty" bson:"conversation_id,omitempty"`
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
	CanceledAt     *time.Time     `json:"canceled_at" bson:"canceled_at"`
	CanceledBy     *ActorRef      `json:"canceled_by" bson:"canceled_by"`
	CancelReason   *string        `json:"cancel_reason" bson:"cancel_reason"`
	CreatedAt      time.Time      `json:"created_at" bson:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at" bson:"updated_at"`
}

func (t *TaskDetail) NextDispatchableTodo() *Todo {
	if t == nil {
		return nil
	}
	for i := range t.Todos {
		todo := &t.Todos[i]
		switch todo.Status {
		case "done":
			continue
		case "pending":
			return todo
		default:
			return nil
		}
	}
	return nil
}

func (t *TaskDetail) CanDispatchTodo(todoID string) bool {
	next := t.NextDispatchableTodo()
	return next != nil && next.ID == todoID
}
