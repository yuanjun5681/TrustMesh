package protocol

import "trustmesh/backend/internal/model"

type WebhookPayload struct {
	NodeID     string         `json:"nodeId"`
	Type       string         `json:"type"`
	From       string         `json:"from"`
	SessionKey string         `json:"sessionKey"`
	Message    string         `json:"message"`
	Metadata   map[string]any `json:"metadata"`
}

type ConversationReplyPayload struct {
	ConversationID string `json:"conversation_id"`
	Content        string `json:"content"`
}

type TaskCreatePayload struct {
	ProjectID      string                  `json:"project_id"`
	ConversationID string                  `json:"conversation_id"`
	Title          string                  `json:"title"`
	Description    string                  `json:"description"`
	Todos          []TaskCreateTodoPayload `json:"todos"`
}

type TaskCreateTodoPayload struct {
	ID             string `json:"id"`
	Title          string `json:"title"`
	Description    string `json:"description"`
	AssigneeNodeID string `json:"assignee_node_id"`
}

type TodoProgressPayload struct {
	TaskID  string `json:"task_id"`
	TodoID  string `json:"todo_id"`
	Message string `json:"message"`
}

type TodoCompletePayload struct {
	TaskID string           `json:"task_id"`
	TodoID string           `json:"todo_id"`
	Result model.TodoResult `json:"result"`
}

type TodoFailPayload struct {
	TaskID string `json:"task_id"`
	TodoID string `json:"todo_id"`
	Error  string `json:"error"`
}

type TaskCommentPayload struct {
	TaskID  string `json:"task_id"`
	TodoID  string `json:"todo_id,omitempty"`
	Content string `json:"content"`
}

type TaskCreatedPayload struct {
	TaskID         string `json:"task_id"`
	ProjectID      string `json:"project_id"`
	ConversationID string `json:"conversation_id"`
	Title          string `json:"title"`
}

type TaskStatusChangedPayload struct {
	TaskID      string `json:"task_id"`
	Status      string `json:"status"`
	ActorNodeID string `json:"actor_node_id,omitempty"`
	Cause       string `json:"cause,omitempty"`
	Version     int    `json:"version"`
}

type TodoExecBrief struct {
	Objective    string `json:"objective"`
	MustUseSkill string `json:"must_use_skill"`
}

type TodoAssignedPayload struct {
	TaskID      string         `json:"task_id"`
	TodoID      string         `json:"todo_id"`
	Title       string         `json:"title"`
	Description string         `json:"description"`
	Content     string         `json:"content"`
	ExecBrief   *TodoExecBrief `json:"exec_brief,omitempty"`
}

type TodoStatusChangedPayload struct {
	TaskID      string `json:"task_id"`
	TodoID      string `json:"todo_id"`
	Status      string `json:"status"`
	ActorNodeID string `json:"actor_node_id,omitempty"`
	Cause       string `json:"cause,omitempty"`
	Version     int    `json:"version"`
	Message     string `json:"message,omitempty"`
}

type PMConversationProject struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

type PMConversationBrief struct {
	Objective                   string `json:"objective"`
	MustClarifyBeforeTaskCreate bool   `json:"must_clarify_before_task_create"`
	MustUseSkill                string `json:"must_use_skill"`
}

type PMConversationAgent struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	NodeID       string   `json:"node_id"`
	Role         string   `json:"role"`
	Status       string   `json:"status"`
	Capabilities []string `json:"capabilities"`
}

type PMConversationMessage struct {
	ConversationID  string                 `json:"conversation_id"`
	ProjectID       string                 `json:"project_id"`
	Content         string                 `json:"content"`
	UserContent     string                 `json:"user_content"`
	IsInitial       bool                   `json:"is_initial_message"`
	Project         *PMConversationProject `json:"project,omitempty"`
	PMBrief         *PMConversationBrief   `json:"pm_brief,omitempty"`
	CandidateAgents []PMConversationAgent  `json:"candidate_agents,omitempty"`
}
