package store

import (
	"time"

	"go.uber.org/zap"
)

func (s *Store) heartbeatLoop() {
	ticker := time.NewTicker(s.heartbeatSweepInterval)
	defer ticker.Stop()
	defer close(s.heartbeatDoneCh)

	for {
		select {
		case <-ticker.C:
			s.ReconcileAgentStatuses(time.Now().UTC())
		case <-s.heartbeatStopCh:
			return
		}
	}
}

func (s *Store) ReconcileAgentStatuses(now time.Time) int {
	if now.IsZero() {
		now = time.Now().UTC()
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	updated := 0
	persisted := make(map[string]struct{})
	for _, agent := range s.agents {
		if agent.HeartbeatAt == nil {
			continue
		}
		if agent.Status == "offline" {
			continue
		}
		if now.Sub(*agent.HeartbeatAt) <= s.heartbeatTTL {
			continue
		}

		agent.Status = "offline"
		agent.UpdatedAt = now
		s.rebuildProjectPMSummariesUnsafe(agent.ID)
		s.rebuildTaskPMSummariesUnsafe(agent.ID)
		if _, ok := persisted[agent.ID]; !ok {
			if err := s.persistAgentGraphUnsafe(agent.ID); err != nil && s.log != nil {
				s.log.Warn("failed to persist offline agent status", zap.String("agent_id", agent.ID), zap.Error(err))
			}
			persisted[agent.ID] = struct{}{}
		}
		updated++
	}
	return updated
}
