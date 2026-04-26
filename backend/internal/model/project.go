package model

import "time"

type ProjectTaskSummary struct {
	TaskTotal       int        `json:"task_total" bson:"task_total"`
	PendingCount    int        `json:"pending_count" bson:"pending_count"`
	InProgressCount int        `json:"in_progress_count" bson:"in_progress_count"`
	DoneCount       int        `json:"done_count" bson:"done_count"`
	FailedCount     int        `json:"failed_count" bson:"failed_count"`
	CanceledCount   int        `json:"canceled_count" bson:"canceled_count"`
	WorkStatus      string     `json:"work_status" bson:"work_status"`
	LatestTaskAt    *time.Time `json:"latest_task_at" bson:"latest_task_at"`
}

type Project struct {
	ID          string             `json:"id" bson:"_id"`
	UserID      string             `json:"-" bson:"user_id"`
	Name        string             `json:"name" bson:"name"`
	Description string             `json:"description" bson:"description"`
	Status      string             `json:"status" bson:"status"`
	TaskSummary ProjectTaskSummary `json:"task_summary" bson:"task_summary"`
	PMAgentID            string             `json:"-" bson:"pm_agent_id"`
	PMAgent              PMAgentSummary     `json:"pm_agent" bson:"pm_agent"`
	SourcePlatform       string             `json:"source_platform,omitempty"         bson:"source_platform,omitempty"`
	SourcePlatformNodeID string             `json:"source_platform_node_id,omitempty" bson:"source_platform_node_id,omitempty"`
	CreatedAt            time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt            time.Time          `json:"updated_at" bson:"updated_at"`
}
