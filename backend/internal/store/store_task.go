package store

import (
	"sort"

	"trustmesh/backend/internal/model"
	"trustmesh/backend/internal/transport"
)

func (s *Store) ListTasks(userID, projectID, status string) ([]model.TaskListItem, *transport.AppError) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if _, err := s.projectForUserUnsafe(userID, projectID); err != nil {
		return nil, err
	}
	if status != "" && !isValidTaskStatus(status) {
		return nil, transport.Validation("invalid status", map[string]any{"status": "must be pending/in_progress/done/failed"})
	}

	ids := s.projectTasks[projectID]
	items := make([]model.TaskListItem, 0, len(ids))
	for _, id := range ids {
		task, ok := s.tasks[id]
		if !ok || task.UserID != userID {
			continue
		}
		if status != "" && task.Status != status {
			continue
		}
		items = append(items, toTaskListItem(*task))
	}
	sort.Slice(items, func(i, j int) bool { return items[i].UpdatedAt.After(items[j].UpdatedAt) })
	return items, nil
}

func (s *Store) GetTask(userID, taskID string) (*model.TaskDetail, *transport.AppError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	task, ok := s.tasks[taskID]
	if !ok || task.UserID != userID {
		return nil, transport.NotFound("task not found")
	}
	return copyTask(task), nil
}

func (s *Store) ListTaskEvents(userID, taskID string) ([]model.Event, *transport.AppError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	task, ok := s.tasks[taskID]
	if !ok || task.UserID != userID {
		return nil, transport.NotFound("task not found")
	}
	events := s.taskEvents[taskID]
	cloned := make([]model.Event, len(events))
	copy(cloned, events)
	sort.Slice(cloned, func(i, j int) bool { return cloned[i].CreatedAt.Before(cloned[j].CreatedAt) })
	return cloned, nil
}

func (s *Store) ListUserEvents(userID string, limit int) []model.Event {
	s.mu.RLock()
	defer s.mu.RUnlock()
	events := s.userEvents[userID]
	if limit <= 0 {
		limit = len(events)
	}
	result := make([]model.Event, 0, min(limit, len(events)))
	for i := len(events) - 1; i >= 0 && len(result) < limit; i-- {
		if events[i].EventType == "agent_status_changed" {
			continue
		}
		result = append(result, *events[i])
	}
	return result
}

func (s *Store) ListAgentEvents(userID, agentID string, limit int) ([]model.Event, *transport.AppError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	agent, ok := s.agents[agentID]
	if !ok || agent.UserID != userID {
		return nil, transport.NotFound("agent not found")
	}
	events := s.agentEvents[agentID]
	if limit <= 0 || limit > len(events) {
		limit = len(events)
	}
	result := make([]model.Event, 0, limit)
	for i := len(events) - 1; i >= 0 && len(result) < limit; i-- {
		result = append(result, *events[i])
	}
	return result, nil
}

func (s *Store) ListRecentTasks(userID string, limit int) []model.TaskListItem {
	s.mu.RLock()
	defer s.mu.RUnlock()

	items := make([]model.TaskListItem, 0)
	for _, t := range s.tasks {
		if t.UserID == userID {
			items = append(items, toTaskListItem(*t))
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].UpdatedAt.After(items[j].UpdatedAt) })
	if limit > 0 && limit < len(items) {
		items = items[:limit]
	}
	return items
}

func (s *Store) getTaskSummaryByConversationUnsafe(conversationID string) *model.TaskSummary {
	taskID, ok := s.conversationTasks[conversationID]
	if !ok {
		return nil
	}
	task, ok := s.tasks[taskID]
	if !ok {
		return nil
	}
	completed := 0
	for _, todo := range task.Todos {
		if todo.Status == "done" {
			completed++
		}
	}
	return &model.TaskSummary{
		ID:                 task.ID,
		Title:              task.Title,
		Status:             task.Status,
		Priority:           task.Priority,
		TodoCount:          len(task.Todos),
		CompletedTodoCount: completed,
		CreatedAt:          task.CreatedAt,
		UpdatedAt:          task.UpdatedAt,
	}
}

func toTaskListItem(task model.TaskDetail) model.TaskListItem {
	completed := 0
	failed := 0
	for _, td := range task.Todos {
		if td.Status == "done" {
			completed++
		}
		if td.Status == "failed" {
			failed++
		}
	}
	return model.TaskListItem{
		ID:                 task.ID,
		ProjectID:          task.ProjectID,
		ConversationID:     task.ConversationID,
		Title:              task.Title,
		Description:        task.Description,
		Status:             task.Status,
		Priority:           task.Priority,
		PMAgent:            task.PMAgent,
		TodoCount:          len(task.Todos),
		CompletedTodoCount: completed,
		FailedTodoCount:    failed,
		CreatedAt:          task.CreatedAt,
		UpdatedAt:          task.UpdatedAt,
	}
}

func toPMSummary(a *model.Agent) model.PMAgentSummary {
	return model.PMAgentSummary{ID: a.ID, Name: a.Name, NodeID: a.NodeID, Status: a.Status}
}

func isValidTaskStatus(status string) bool {
	switch status {
	case "pending", "in_progress", "done", "failed":
		return true
	default:
		return false
	}
}
