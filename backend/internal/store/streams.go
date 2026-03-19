package store

import (
	"trustmesh/backend/internal/model"
	"trustmesh/backend/internal/transport"
)

const streamBufferSize = 4

func (s *Store) SubscribeTask(userID, taskID string) (<-chan model.TaskStreamSnapshot, func(), *transport.AppError) {
	s.mu.RLock()
	task, ok := s.tasks[taskID]
	if !ok || task.UserID != userID {
		s.mu.RUnlock()
		return nil, nil, transport.NotFound("task not found")
	}
	snapshot := s.taskSnapshotUnsafe(taskID)
	s.mu.RUnlock()

	ch := make(chan model.TaskStreamSnapshot, streamBufferSize)

	s.streamMu.Lock()
	if s.taskSubscribers[taskID] == nil {
		s.taskSubscribers[taskID] = make(map[chan model.TaskStreamSnapshot]struct{})
	}
	s.taskSubscribers[taskID][ch] = struct{}{}
	s.streamMu.Unlock()

	sendLatestTaskSnapshot(ch, snapshot)

	return ch, func() {
		s.streamMu.Lock()
		if subscribers, ok := s.taskSubscribers[taskID]; ok {
			delete(subscribers, ch)
			if len(subscribers) == 0 {
				delete(s.taskSubscribers, taskID)
			}
		}
		s.streamMu.Unlock()
		close(ch)
	}, nil
}

func (s *Store) SubscribeConversation(userID, conversationID string) (<-chan model.ConversationStreamSnapshot, func(), *transport.AppError) {
	s.mu.RLock()
	conv, ok := s.conversations[conversationID]
	if !ok || conv.UserID != userID {
		s.mu.RUnlock()
		return nil, nil, transport.NotFound("conversation not found")
	}
	snapshot := s.conversationSnapshotUnsafe(conversationID)
	s.mu.RUnlock()

	ch := make(chan model.ConversationStreamSnapshot, streamBufferSize)

	s.streamMu.Lock()
	if s.conversationSubscribers[conversationID] == nil {
		s.conversationSubscribers[conversationID] = make(map[chan model.ConversationStreamSnapshot]struct{})
	}
	s.conversationSubscribers[conversationID][ch] = struct{}{}
	s.streamMu.Unlock()

	sendLatestConversationSnapshot(ch, snapshot)

	return ch, func() {
		s.streamMu.Lock()
		if subscribers, ok := s.conversationSubscribers[conversationID]; ok {
			delete(subscribers, ch)
			if len(subscribers) == 0 {
				delete(s.conversationSubscribers, conversationID)
			}
		}
		s.streamMu.Unlock()
		close(ch)
	}, nil
}

func (s *Store) publishTaskUnsafe(taskID string) {
	snapshot := s.taskSnapshotUnsafe(taskID)

	s.streamMu.RLock()
	defer s.streamMu.RUnlock()

	for ch := range s.taskSubscribers[taskID] {
		sendLatestTaskSnapshot(ch, snapshot)
	}
}

func (s *Store) publishConversationUnsafe(conversationID string) {
	snapshot := s.conversationSnapshotUnsafe(conversationID)

	s.streamMu.RLock()
	defer s.streamMu.RUnlock()

	for ch := range s.conversationSubscribers[conversationID] {
		sendLatestConversationSnapshot(ch, snapshot)
	}
}

func (s *Store) taskSnapshotUnsafe(taskID string) model.TaskStreamSnapshot {
	task := copyTask(s.tasks[taskID])
	events := append([]model.Event(nil), s.taskEvents[taskID]...)
	return model.TaskStreamSnapshot{
		Task:   *task,
		Events: events,
	}
}

func (s *Store) conversationSnapshotUnsafe(conversationID string) model.ConversationStreamSnapshot {
	return model.ConversationStreamSnapshot{
		Conversation: s.toConversationDetailUnsafe(s.conversations[conversationID]),
	}
}

func sendLatestTaskSnapshot(ch chan model.TaskStreamSnapshot, snapshot model.TaskStreamSnapshot) {
	select {
	case ch <- snapshot:
		return
	default:
	}

	select {
	case <-ch:
	default:
	}

	select {
	case ch <- snapshot:
	default:
	}
}

func sendLatestConversationSnapshot(ch chan model.ConversationStreamSnapshot, snapshot model.ConversationStreamSnapshot) {
	select {
	case ch <- snapshot:
		return
	default:
	}

	select {
	case <-ch:
	default:
	}

	select {
	case ch <- snapshot:
	default:
	}
}
