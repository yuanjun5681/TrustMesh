package model

import "time"

type PMAgentSummary struct {
	ID     string `json:"id" bson:"id"`
	Name   string `json:"name" bson:"name"`
	NodeID string `json:"node_id" bson:"node_id"`
	Status string `json:"status" bson:"status"`
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
