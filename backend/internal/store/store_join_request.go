package store

import (
	"sort"
	"strings"
	"time"

	"trustmesh/backend/internal/model"
	"trustmesh/backend/internal/transport"
)

type CreateJoinRequestInput struct {
	TrustRequestID string
	NodeID         string
	Name           string
	Description    string
	Role           string
	Capabilities   []string
	AgentProduct   string
	ReceivedAt     time.Time
}

type JoinRequestOverrides struct {
	Name         *string  `json:"name,omitempty"`
	Role         *string  `json:"role,omitempty"`
	Description  *string  `json:"description,omitempty"`
	Capabilities []string `json:"capabilities,omitempty"`
}

// HasTrustRequest checks if a trust request ID has already been processed (lock-free read).
func (s *Store) HasTrustRequest(trustRequestID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, exists := s.trustRequestIndex[trustRequestID]
	return exists
}

func (s *Store) CreateJoinRequest(in CreateJoinRequestInput) (*model.JoinRequest, *transport.AppError) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Dedup by trust request ID
	if _, exists := s.trustRequestIndex[in.TrustRequestID]; exists {
		return nil, transport.Conflict("JOIN_REQUEST_EXISTS", "join request already exists for this trust request")
	}

	// Check if node is already registered as an agent
	if _, exists := s.agentByNode[in.NodeID]; exists {
		return nil, transport.Conflict("AGENT_NODE_ID_EXISTS", "node_id already registered as an agent")
	}

	// Check if there's already a pending request from this node
	for _, jr := range s.joinRequests {
		if jr.NodeID == in.NodeID && jr.Status == "pending" {
			return nil, transport.Conflict("JOIN_REQUEST_PENDING", "a pending join request already exists for this node")
		}
	}

	name := strings.TrimSpace(in.Name)
	if name == "" {
		name = in.NodeID
	}
	role := strings.TrimSpace(in.Role)
	if role == "" || !isValidRole(role) {
		role = "custom"
	}

	now := in.ReceivedAt
	if now.IsZero() {
		now = time.Now().UTC()
	}

	jr := &model.JoinRequest{
		ID:             newID(),
		TrustRequestID: in.TrustRequestID,
		NodeID:         in.NodeID,
		Name:           name,
		Description:    strings.TrimSpace(in.Description),
		Role:           role,
		Capabilities:   normalizeCapabilities(in.Capabilities),
		AgentProduct:   strings.TrimSpace(in.AgentProduct),
		Status:         "pending",
		Metadata:       map[string]any{},
		CreatedAt:      now,
	}

	s.joinRequests[jr.ID] = jr
	s.trustRequestIndex[in.TrustRequestID] = jr.ID

	// Associate with all users (for now, all users can see join requests)
	for _, u := range s.users {
		s.userJoinRequests[u.ID] = append(s.userJoinRequests[u.ID], jr.ID)
	}


	// Notify all users
	for _, u := range s.users {
		content := "Agent「" + jr.Name + "」申请加入平台"
		event := &model.Event{
			ID:        newID(),
			UserID:    u.ID,
			EventType: "join_request_received",
			ActorType: "agent",
			ActorID:   jr.NodeID,
			ActorName: jr.Name,
			Content:   &content,
			Metadata:  map[string]any{"join_request_id": jr.ID, "node_id": jr.NodeID},
			CreatedAt: now,
		}
		s.maybeCreateNotificationUnsafe(event)
		s.publishUserEventUnsafe(u.ID, "join_request.created", map[string]any{
			"join_request": *jr,
		}, now)
	}

	clone := *jr
	return &clone, nil
}

func (s *Store) ListJoinRequests(userID, status string) []model.JoinRequest {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ids := s.userJoinRequests[userID]
	items := make([]model.JoinRequest, 0, len(ids))
	for _, id := range ids {
		jr, ok := s.joinRequests[id]
		if !ok {
			continue
		}
		if status != "" && jr.Status != status {
			continue
		}
		items = append(items, *jr)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt.After(items[j].CreatedAt) })
	return items
}

func (s *Store) GetJoinRequest(userID, requestID string) (*model.JoinRequest, *transport.AppError) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	jr, ok := s.joinRequests[requestID]
	if !ok {
		return nil, transport.NotFound("join request not found")
	}
	clone := *jr
	return &clone, nil
}

func (s *Store) ApproveJoinRequest(userID, requestID string, overrides JoinRequestOverrides) (*model.Agent, *transport.AppError) {
	s.mu.Lock()
	defer s.mu.Unlock()

	jr, ok := s.joinRequests[requestID]
	if !ok {
		return nil, transport.NotFound("join request not found")
	}
	if jr.Status != "pending" {
		return nil, transport.Validation("join request is not pending", map[string]any{"status": jr.Status})
	}

	// Check node_id not taken by active agent
	if _, exists := s.agentByNode[jr.NodeID]; exists {
		return nil, transport.Conflict("AGENT_NODE_ID_EXISTS", "node_id already registered as an agent")
	}

	// Apply overrides
	name := jr.Name
	if overrides.Name != nil && strings.TrimSpace(*overrides.Name) != "" {
		name = strings.TrimSpace(*overrides.Name)
	}
	role := jr.Role
	if overrides.Role != nil && isValidRole(strings.TrimSpace(*overrides.Role)) {
		role = strings.TrimSpace(*overrides.Role)
	}
	description := jr.Description
	if overrides.Description != nil && strings.TrimSpace(*overrides.Description) != "" {
		description = strings.TrimSpace(*overrides.Description)
	}
	capabilities := normalizeCapabilities(jr.Capabilities)
	if overrides.Capabilities != nil {
		capabilities = normalizeCapabilities(overrides.Capabilities)
	}

	now := time.Now().UTC()

	// Check if there's an archived agent with the same node_id — restore it instead of creating a new one
	var agent *model.Agent
	for _, a := range s.agents {
		if a.NodeID == jr.NodeID && a.Archived {
			a.Name = name
			a.Description = description
			a.Role = role
			a.Capabilities = capabilities
			a.Archived = false
			a.Status = "offline"
			a.UpdatedAt = now
			agent = a
			break
		}
	}

	if agent == nil {
		// Create new agent
		agent = &model.Agent{
			ID:           newID(),
			UserID:       userID,
			Name:         name,
			Description:  description,
			Role:         role,
			Capabilities: capabilities,
			NodeID:       jr.NodeID,
			Status:       "offline",
			CreatedAt:    now,
			UpdatedAt:    now,
		}
		s.agents[agent.ID] = agent
	}

	s.agentByNode[jr.NodeID] = agent.ID
	if err := s.persistAgentUnsafe(agent); err != nil {
		return nil, mongoWriteError(err)
	}

	// Mark join request as approved
	jr.Status = "approved"
	jr.AgentID = agent.ID
	jr.UserID = userID
	resolvedAt := now
	jr.ResolvedAt = &resolvedAt

	clone := copyAgent(agent)
	clone.Usage = s.agentUsageUnsafe(agent.ID)
	return clone, nil
}

func (s *Store) RejectJoinRequest(userID, requestID string) *transport.AppError {
	s.mu.Lock()
	defer s.mu.Unlock()

	jr, ok := s.joinRequests[requestID]
	if !ok {
		return transport.NotFound("join request not found")
	}
	if jr.Status != "pending" {
		return transport.Validation("join request is not pending", map[string]any{"status": jr.Status})
	}

	now := time.Now().UTC()
	jr.Status = "rejected"
	jr.UserID = userID
	jr.ResolvedAt = &now
	return nil
}

func (s *Store) PendingJoinRequestCount(userID string) int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	count := 0
	for _, id := range s.userJoinRequests[userID] {
		if jr, ok := s.joinRequests[id]; ok && jr.Status == "pending" {
			count++
		}
	}
	return count
}
