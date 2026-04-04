package store

import (
	"go.uber.org/zap"
)

func (s *Store) publishTaskUnsafe(taskID string) {
	task := s.copyTaskWithArtifactsUnsafe(s.tasks[taskID])
	s.publishUserEventUnsafe(task.UserID, "task.updated", map[string]any{
		"task": *task,
	}, task.UpdatedAt)

	if s.log != nil {
		s.log.Info("publishTaskUnsafe",
			zap.String("task_id", taskID),
			zap.String("status", string(task.Status)),
		)
	}
}
func (s *Store) publishAgentChatUnsafe(chatID string) {
	chat, ok := s.agentChats[chatID]
	if !ok || chat.Status != "active" {
		return
	}
	detail := s.toAgentChatDetailUnsafe(chat)
	s.publishUserEventUnsafe(chat.UserID, "agent_chat.updated", map[string]any{
		"chat": detail,
	}, detail.UpdatedAt)

	if s.log != nil {
		s.log.Info("publishAgentChatUnsafe",
			zap.String("chat_id", chatID),
			zap.String("agent_id", chat.AgentID),
			zap.Int("messages", len(detail.Messages)),
		)
	}
}
