package model

import "time"

type PlatformConnection struct {
	ID             string    `bson:"_id"              json:"id"`
	UserID         string    `bson:"user_id"          json:"user_id"`
	Platform       string    `bson:"platform"         json:"platform"`           // e.g. "clawhire"
	PlatformNodeID string    `bson:"platform_node_id" json:"platform_node_id"`   // ClawSynapse nodeId of remote platform
	RemoteUserID   string    `bson:"remote_user_id"   json:"remote_user_id"`     // user ID on remote platform
	PMAgentID      string    `bson:"pm_agent_id"      json:"pm_agent_id"`        // local PM agent for this binding
	LinkedAt       time.Time `bson:"linked_at"        json:"linked_at"`
}

type ExternalTaskRef struct {
	Platform       string `bson:"platform"         json:"platform"`
	ExternalTaskID string `bson:"external_task_id" json:"external_task_id"`
	RemoteUserID   string `bson:"remote_user_id"   json:"remote_user_id"`
	PlatformNodeID string `bson:"platform_node_id" json:"platform_node_id"`
	ContractID     string `bson:"contract_id,omitempty" json:"contract_id,omitempty"`
}
