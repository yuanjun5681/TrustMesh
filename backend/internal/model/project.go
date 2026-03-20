package model

import "time"

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
