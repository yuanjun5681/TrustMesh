package store

import "trustmesh/backend/internal/model"

func (s *Store) maybeCreateNotificationUnsafe(event *model.Event) {
	var title, body, category, priority string
	switch event.EventType {
	case "task_created":
		title = "任务已创建"
		body = stringOrDefault(event.Content, "新任务")
		category = "task"
		priority = "medium"
	case "task_status_changed":
		title = "任务状态变更"
		body = stringOrDefault(event.Content, "状态已变更")
		category = "task"
		priority = "high"
	case "todo_assigned":
		title = "Todo 已分派"
		body = stringOrDefault(event.Content, "新的 Todo 分派")
		category = "todo"
		priority = "low"
	case "todo_started":
		title = "Todo 开始执行"
		body = stringOrDefault(event.Content, "Todo 已开始")
		category = "todo"
		priority = "low"
	case "todo_completed":
		title = "Todo 已完成"
		body = stringOrDefault(event.Content, "Todo 完成")
		category = "todo"
		priority = "medium"
	case "todo_failed":
		title = "Todo 执行失败"
		body = stringOrDefault(event.Content, "Todo 失败")
		category = "todo"
		priority = "high"
	case "conversation_reply":
		title = "PM 回复"
		body = stringOrDefault(event.Content, "新的回复")
		category = "conversation"
		priority = "medium"
	default:
		return
	}

	var conversationID string
	if cid, ok := event.Metadata["conversation_id"].(string); ok && cid != "" {
		conversationID = cid
	} else if event.TaskID != "" {
		if task, ok := s.tasks[event.TaskID]; ok {
			conversationID = task.ConversationID
		}
	}

	now := event.CreatedAt
	notification := &model.Notification{
		ID:             newID(),
		UserID:         event.UserID,
		EventID:        event.ID,
		ProjectID:      event.ProjectID,
		TaskID:         event.TaskID,
		ConversationID: conversationID,
		Title:          title,
		Body:           body,
		Category:       category,
		Priority:       priority,
		IsRead:         false,
		ReadAt:         nil,
		CreatedAt:      now,
	}
	s.notifications[notification.ID] = notification
	s.userNotifications[event.UserID] = append(s.userNotifications[event.UserID], notification.ID)
	s.persistNotificationUnsafe(notification)
}

func stringOrDefault(s *string, def string) string {
	if s != nil && *s != "" {
		return *s
	}
	return def
}
