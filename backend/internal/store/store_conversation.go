package store

import (
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"trustmesh/backend/internal/model"
	"trustmesh/backend/internal/transport"
)

func (s *Store) CreateConversation(userID, projectID, content string) (*model.ConversationDetail, *transport.AppError) {
	content = strings.TrimSpace(content)
	if content == "" {
		return nil, transport.Validation("invalid content", map[string]any{"content": "required"})
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	project, err := s.projectForUserUnsafe(userID, projectID)
	if err != nil {
		return nil, err
	}
	if project.Status == "archived" {
		return nil, transport.Conflict("PROJECT_ARCHIVED", "archived project cannot create conversations")
	}

	if err := s.validateProjectPMAgentOnlineUnsafe(project); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	msg := model.ConversationMessage{ID: uuid.NewString(), Role: "user", Content: content, CreatedAt: now}
	conv := &model.Conversation{
		ID:        newID(),
		UserID:    userID,
		ProjectID: projectID,
		Status:    "active",
		Messages:  []model.ConversationMessage{msg},
		CreatedAt: now,
		UpdatedAt: now,
	}
	s.conversations[conv.ID] = conv
	s.projectConversations[projectID] = append(s.projectConversations[projectID], conv.ID)
	if err := s.persistConversationUnsafe(conv); err != nil {
		return nil, mongoWriteError(err)
	}
	s.publishConversationUnsafe(conv.ID)

	detail := s.toConversationDetailUnsafe(conv)
	return &detail, nil
}

func (s *Store) ListConversations(userID, projectID string) ([]model.ConversationListItem, *transport.AppError) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if _, err := s.projectForUserUnsafe(userID, projectID); err != nil {
		return nil, err
	}

	ids := s.projectConversations[projectID]
	items := make([]model.ConversationListItem, 0, len(ids))
	for _, id := range ids {
		conv, ok := s.conversations[id]
		if !ok || conv.UserID != userID {
			continue
		}
		if len(conv.Messages) == 0 {
			continue
		}
		last := conv.Messages[len(conv.Messages)-1]
		item := model.ConversationListItem{
			ID:          conv.ID,
			ProjectID:   conv.ProjectID,
			Status:      conv.Status,
			LastMessage: last,
			LinkedTask:  s.getTaskSummaryByConversationUnsafe(conv.ID),
			CreatedAt:   conv.CreatedAt,
			UpdatedAt:   conv.UpdatedAt,
		}
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].UpdatedAt.After(items[j].UpdatedAt) })
	return items, nil
}

func (s *Store) GetConversation(userID, conversationID string) (*model.ConversationDetail, *transport.AppError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	conv, ok := s.conversations[conversationID]
	if !ok || conv.UserID != userID {
		return nil, transport.NotFound("conversation not found")
	}
	detail := s.toConversationDetailUnsafe(conv)
	return &detail, nil
}

func (s *Store) AppendConversationMessage(userID, conversationID, content string) (*model.ConversationDetail, *transport.AppError) {
	content = strings.TrimSpace(content)
	if content == "" {
		return nil, transport.Validation("invalid content", map[string]any{"content": "required"})
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	conv, ok := s.conversations[conversationID]
	if !ok || conv.UserID != userID {
		return nil, transport.NotFound("conversation not found")
	}
	if conv.Status != "active" {
		return nil, transport.Conflict("CONVERSATION_RESOLVED", "conversation is resolved")
	}
	project, ok := s.projects[conv.ProjectID]
	if !ok {
		return nil, transport.NotFound("project not found")
	}
	if err := s.validateProjectPMAgentOnlineUnsafe(project); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	conv.Messages = append(conv.Messages, model.ConversationMessage{ID: uuid.NewString(), Role: "user", Content: content, CreatedAt: now})
	conv.UpdatedAt = now
	if err := s.persistConversationUnsafe(conv); err != nil {
		return nil, mongoWriteError(err)
	}
	s.publishConversationUnsafe(conv.ID)
	detail := s.toConversationDetailUnsafe(conv)
	return &detail, nil
}

func (s *Store) AppendPMReplyByNode(nodeID, conversationID, content string) (*model.ConversationDetail, *transport.AppError) {
	conversationID = strings.TrimSpace(conversationID)
	content = strings.TrimSpace(content)
	if conversationID == "" || content == "" {
		return nil, transport.Validation("invalid conversation reply payload", map[string]any{"conversation_id": "required", "content": "required"})
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	pmAgent, err := s.agentByNodeUnsafe(nodeID)
	if err != nil {
		return nil, err
	}
	if pmAgent.Role != "pm" {
		return nil, transport.Forbidden("only pm agent can reply conversation")
	}

	conv, ok := s.conversations[conversationID]
	if !ok {
		return nil, transport.NotFound("conversation not found")
	}
	if conv.Status != "active" {
		return nil, transport.Conflict("CONVERSATION_RESOLVED", "conversation is resolved")
	}

	project, ok := s.projects[conv.ProjectID]
	if !ok {
		return nil, transport.NotFound("project not found")
	}
	if project.PMAgentID != pmAgent.ID {
		return nil, transport.Forbidden("pm agent is not bound to this project")
	}

	now := time.Now().UTC()
	s.markAgentSeenUnsafe(pmAgent.ID, now)
	conv.Messages = append(conv.Messages, model.ConversationMessage{ID: uuid.NewString(), Role: "pm_agent", Content: content, CreatedAt: now})
	conv.UpdatedAt = now

	s.addEventUnsafe(conv.UserID, conv.ProjectID, "", "", "agent", pmAgent.ID, pmAgent.Name, "conversation_reply", &content, map[string]any{
		"conversation_id": conv.ID,
	}, now)

	if err := s.persistConversationUnsafe(conv); err != nil {
		return nil, mongoWriteError(err)
	}
	if err := s.persistAgentGraphUnsafe(pmAgent.ID); err != nil {
		return nil, mongoWriteError(err)
	}
	s.publishConversationUnsafe(conv.ID)
	detail := s.toConversationDetailUnsafe(conv)
	return &detail, nil
}

func (s *Store) toConversationDetailUnsafe(conv *model.Conversation) model.ConversationDetail {
	messages := make([]model.ConversationMessage, len(conv.Messages))
	copy(messages, conv.Messages)
	return model.ConversationDetail{
		ID:         conv.ID,
		ProjectID:  conv.ProjectID,
		Status:     conv.Status,
		Messages:   messages,
		LinkedTask: s.getTaskSummaryByConversationUnsafe(conv.ID),
		CreatedAt:  conv.CreatedAt,
		UpdatedAt:  conv.UpdatedAt,
	}
}
