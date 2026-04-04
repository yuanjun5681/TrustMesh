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
