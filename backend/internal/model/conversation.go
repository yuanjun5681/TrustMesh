package model

import "time"

type ConversationMessage struct {
	ID        string    `json:"id" bson:"id"`
	Role      string    `json:"role" bson:"role"`
	Content   string    `json:"content" bson:"content"`
	CreatedAt time.Time `json:"created_at" bson:"created_at"`
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
