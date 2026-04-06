package store

import (
	"testing"
	"time"
)

func TestResetAgentChatInitializesEmptyMessages(t *testing.T) {
	s := New()
	user, appErr := s.CreateUser("user@example.com", "User", "hash")
	if appErr != nil {
		t.Fatalf("create user: %v", appErr)
	}
	agent, appErr := s.CreateAgent(user.ID, "node-chat-000", "Chat Agent", "developer", "dev", []string{"conversation"})
	if appErr != nil {
		t.Fatalf("create agent: %v", appErr)
	}
	now := time.Now().UTC()
	s.SyncAgentPresence([]AgentPresence{{NodeID: agent.NodeID, LastSeenAt: now}}, now)

	chat, appErr := s.ResetAgentChat(user.ID, agent.ID)
	if appErr != nil {
		t.Fatalf("reset chat: %v", appErr)
	}
	if chat.Messages == nil {
		t.Fatal("expected empty messages slice, got nil")
	}
	if len(chat.Messages) != 0 {
		t.Fatalf("expected empty messages slice, got %d messages", len(chat.Messages))
	}

	stored, ok := s.agentChats[chat.ID]
	if !ok {
		t.Fatalf("expected chat %s in store", chat.ID)
	}
	if stored.Messages == nil {
		t.Fatal("expected persisted store chat messages slice, got nil")
	}
}

func TestResetAgentChatRequiresChatCapableOnlineAgent(t *testing.T) {
	s := New()
	user, appErr := s.CreateUser("user@example.com", "User", "hash")
	if appErr != nil {
		t.Fatalf("create user: %v", appErr)
	}

	offlineAgent, appErr := s.CreateAgent(user.ID, "node-offline-001", "Offline", "developer", "dev", []string{"conversation"})
	if appErr != nil {
		t.Fatalf("create offline agent: %v", appErr)
	}
	if _, appErr := s.ResetAgentChat(user.ID, offlineAgent.ID); appErr == nil || appErr.Code != "AGENT_OFFLINE" {
		t.Fatalf("expected AGENT_OFFLINE, got %v", appErr)
	}
}

func TestResetAgentChatRemovesOldSessionRouting(t *testing.T) {
	s := New()
	user, appErr := s.CreateUser("user@example.com", "User", "hash")
	if appErr != nil {
		t.Fatalf("create user: %v", appErr)
	}
	agent, appErr := s.CreateAgent(user.ID, "node-chat-001", "Chat Agent", "developer", "dev", []string{"conversation"})
	if appErr != nil {
		t.Fatalf("create agent: %v", appErr)
	}
	now := time.Now().UTC()
	s.SyncAgentPresence([]AgentPresence{{NodeID: agent.NodeID, LastSeenAt: now}}, now)

	first, appErr := s.ResetAgentChat(user.ID, agent.ID)
	if appErr != nil {
		t.Fatalf("reset first chat: %v", appErr)
	}
	second, appErr := s.ResetAgentChat(user.ID, agent.ID)
	if appErr != nil {
		t.Fatalf("reset second chat: %v", appErr)
	}
	if first.SessionKey == second.SessionKey {
		t.Fatal("expected a new session key after reset")
	}
	if _, ok := s.agentChatBySession[first.SessionKey]; ok {
		t.Fatal("expected old session key to be removed from routing table")
	}
	if _, appErr := s.AppendAgentChatMessageByNode(agent.NodeID, first.SessionKey, "stale reply", "remote-1"); appErr == nil || appErr.Code != "NOT_FOUND" {
		t.Fatalf("expected NOT_FOUND for stale session, got %v", appErr)
	}
}

func TestListAgentChatSessionsReturnsNewestFirst(t *testing.T) {
	s := New()
	user, appErr := s.CreateUser("user@example.com", "User", "hash")
	if appErr != nil {
		t.Fatalf("create user: %v", appErr)
	}
	agent, appErr := s.CreateAgent(user.ID, "node-chat-002", "Chat Agent", "developer", "dev", []string{"conversation"})
	if appErr != nil {
		t.Fatalf("create agent: %v", appErr)
	}
	now := time.Now().UTC()
	s.SyncAgentPresence([]AgentPresence{{NodeID: agent.NodeID, LastSeenAt: now}}, now)

	first, appErr := s.ResetAgentChat(user.ID, agent.ID)
	if appErr != nil {
		t.Fatalf("reset first chat: %v", appErr)
	}
	if _, _, appErr := s.AppendAgentChatUserMessage(user.ID, agent.ID, "older"); appErr != nil {
		t.Fatalf("append first message: %v", appErr)
	}

	second, appErr := s.ResetAgentChat(user.ID, agent.ID)
	if appErr != nil {
		t.Fatalf("reset second chat: %v", appErr)
	}
	if _, _, appErr := s.AppendAgentChatUserMessage(user.ID, agent.ID, "newer"); appErr != nil {
		t.Fatalf("append second message: %v", appErr)
	}

	sessions, appErr := s.ListAgentChatSessions(user.ID, agent.ID)
	if appErr != nil {
		t.Fatalf("list sessions: %v", appErr)
	}
	if len(sessions) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(sessions))
	}
	if sessions[0].ID != second.ID {
		t.Fatalf("expected newest session first, got %s", sessions[0].ID)
	}
	if sessions[0].LastMessagePreview != "newer" {
		t.Fatalf("unexpected preview: %q", sessions[0].LastMessagePreview)
	}
	if sessions[0].MessageCount != 1 {
		t.Fatalf("expected message_count=1, got %d", sessions[0].MessageCount)
	}
	if sessions[1].ID != first.ID {
		t.Fatalf("expected older session second, got %s", sessions[1].ID)
	}
	if sessions[1].Status != "closed" {
		t.Fatalf("expected first session closed, got %s", sessions[1].Status)
	}
}
