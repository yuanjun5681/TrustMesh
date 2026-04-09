package model

import "time"

type JoinRequest struct {
	ID                       string         `json:"id" bson:"_id"`
	UserID                   string         `json:"-" bson:"user_id"`
	TrustRequestID           string         `json:"trust_request_id" bson:"trust_request_id"`
	NodeID                   string         `json:"node_id" bson:"node_id"`
	Name                     string         `json:"name" bson:"name"`
	Description              string         `json:"description" bson:"description"`
	Role                     string         `json:"role" bson:"role"`
	Capabilities             []string       `json:"capabilities" bson:"capabilities"`
	AgentProduct             string         `json:"agent_product" bson:"agent_product"`
	Status                   string         `json:"status" bson:"status"` // pending | approved | rejected
	Metadata                 map[string]any `json:"metadata,omitempty" bson:"metadata,omitempty"`
	ClawAgentID              string         `json:"claw_agent_id,omitempty" bson:"claw_agent_id,omitempty"`
	ApprovedTrustMeshAgentID string         `json:"approved_trustmesh_agent_id,omitempty" bson:"approved_trustmesh_agent_id,omitempty"`
	CreatedAt                time.Time      `json:"created_at" bson:"created_at"`
	ResolvedAt               *time.Time     `json:"resolved_at,omitempty" bson:"resolved_at,omitempty"`
}
