package store

import (
	"time"

	"trustmesh/backend/internal/model"
)

const userStreamBufferSize = 16

func (s *Store) SubscribeUser(userID string) (<-chan model.UserStreamEvent, func()) {
	ch := make(chan model.UserStreamEvent, userStreamBufferSize)

	s.streamMu.Lock()
	if s.userSubscribers[userID] == nil {
		s.userSubscribers[userID] = make(map[chan model.UserStreamEvent]struct{})
	}
	s.userSubscribers[userID][ch] = struct{}{}
	s.streamMu.Unlock()

	return ch, func() {
		s.streamMu.Lock()
		if subscribers, ok := s.userSubscribers[userID]; ok {
			delete(subscribers, ch)
			if len(subscribers) == 0 {
				delete(s.userSubscribers, userID)
			}
		}
		s.streamMu.Unlock()
		close(ch)
	}
}

func (s *Store) publishUserEventUnsafe(userID, eventType string, payload map[string]any, at time.Time) {
	if userID == "" {
		return
	}

	event := model.UserStreamEvent{
		ID:         newID(),
		Type:       eventType,
		OccurredAt: at.UTC(),
		Payload:    copyMap(payload),
	}

	s.streamMu.RLock()
	defer s.streamMu.RUnlock()

	for ch := range s.userSubscribers[userID] {
		sendLatestUserEvent(ch, event)
	}
}

func sendLatestUserEvent(ch chan model.UserStreamEvent, event model.UserStreamEvent) {
	select {
	case ch <- event:
		return
	default:
	}

	select {
	case <-ch:
	default:
	}

	select {
	case ch <- event:
	default:
	}
}
