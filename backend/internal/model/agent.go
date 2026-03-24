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

type AgentStats struct {
	Role string `json:"role"`

	// 执行者维度（developer/reviewer）
	TodosTotal          int      `json:"todos_total"`
	TodosDone           int      `json:"todos_done"`
	TodosFailed         int      `json:"todos_failed"`
	TodosInProgress     int      `json:"todos_in_progress"`
	TodosPending        int      `json:"todos_pending"`
	SuccessRate         float64  `json:"success_rate"`
	AvgResponseTimeMs   *float64 `json:"avg_response_time_ms"`
	AvgCompletionTimeMs *float64 `json:"avg_completion_time_ms"`

	// PM 维度
	ProjectsManaged    int     `json:"projects_managed"`
	TasksCreated       int     `json:"tasks_created"`
	TasksDone          int     `json:"tasks_done"`
	TasksFailed        int     `json:"tasks_failed"`
	TasksInProgress    int     `json:"tasks_in_progress"`
	TasksPending       int     `json:"tasks_pending"`
	TaskSuccessRate    float64 `json:"task_success_rate"`
	ConversationReplies int    `json:"conversation_replies"`

	DailyActivity   []DailyActivityItem `json:"daily_activity"`
	CurrentWorkload []WorkloadItem      `json:"current_workload"`
}

type DailyActivityItem struct {
	Date      string `json:"date"`
	Completed int    `json:"completed"`
	Failed    int    `json:"failed"`
	Created   int    `json:"created"`
}

type WorkloadItem struct {
	TodoID    string `json:"todo_id"`
	TodoTitle string `json:"todo_title"`
	TaskID    string `json:"task_id"`
	TaskTitle string `json:"task_title"`
	ProjectID string `json:"project_id"`
	StartedAt string `json:"started_at"`
}
