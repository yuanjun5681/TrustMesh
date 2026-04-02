package store

import "trustmesh/backend/internal/model"

func (s *Store) maybeCreateNotificationUnsafe(event *model.Event) {
	var title, body, category, priority string
	switch event.EventType {
	case "task_status_changed":
		// 只通知终态（done / failed / canceled），中间状态不通知
		if to, ok := event.Metadata["to"].(string); ok {
			switch to {
			case "done":
				title = "任务已完成"
				body = stringOrDefault(event.Content, "任务完成")
				priority = "medium"
			case "failed":
				title = "任务执行失败"
				body = stringOrDefault(event.Content, "任务失败")
				priority = "high"
			case "canceled":
				title = "任务已取消"
				body = stringOrDefault(event.Content, "任务被取消")
				priority = "medium"
			default:
				return
			}
		} else {
			return
		}
		category = "task"
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
	case "join_request_received":
		title = "Agent 加入申请"
		body = stringOrDefault(event.Content, "收到新的 Agent 加入申请")
		category = "agent"
		priority = "high"
	default:
		return
	}

	// 用户自己触发的操作不通知自己
	if event.ActorType == "user" && event.ActorID == event.UserID {
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
		ActorType:      event.ActorType,
		ActorName:      event.ActorName,
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
	s.publishUserEventUnsafe(event.UserID, "notification.created", map[string]any{
		"notification": *notification,
		"unread_count": unreadNotificationCountUnsafe(s.notifications, s.userNotifications[event.UserID]),
	}, now)
}

func stringOrDefault(s *string, def string) string {
	if s != nil && *s != "" {
		return *s
	}
	return def
}
