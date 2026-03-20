package model

type DashboardStats struct {
	AgentsOnline     int     `json:"agents_online"`
	AgentsTotal      int     `json:"agents_total"`
	TasksInProgress  int     `json:"tasks_in_progress"`
	TasksTotal       int     `json:"tasks_total"`
	TasksDoneCount   int     `json:"tasks_done_count"`
	TasksFailedCount int     `json:"tasks_failed_count"`
	SuccessRate      float64 `json:"success_rate"`
	TodosPending     int     `json:"todos_pending"`
}
