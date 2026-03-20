package store

import (
	"sort"
	"strings"
	"time"

	"trustmesh/backend/internal/model"
	"trustmesh/backend/internal/transport"
)

type UpdateAgentInput struct {
	Name         *string
	Role         *string
	Description  *string
	Capabilities *[]string
}

func (s *Store) CreateAgent(userID, nodeID, name, role, description string, capabilities []string) (*model.Agent, *transport.AppError) {
	nodeID = strings.TrimSpace(nodeID)
	name = strings.TrimSpace(name)
	role = strings.TrimSpace(role)
	description = strings.TrimSpace(description)
	if nodeID == "" || name == "" || role == "" || description == "" {
		return nil, transport.Validation("invalid agent payload", map[string]any{
			"node_id":     "required",
			"name":        "required",
			"role":        "required",
			"description": "required",
		})
	}
	if !isValidRole(role) {
		return nil, transport.Validation("invalid role", map[string]any{"role": "must be one of pm/developer/reviewer/custom"})
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.agentByNode[nodeID]; exists {
		return nil, transport.Conflict("AGENT_NODE_ID_EXISTS", "node_id already exists")
	}

	now := time.Now().UTC()
	agent := &model.Agent{
		ID:           newID(),
		UserID:       userID,
		Name:         name,
		Description:  description,
		Role:         role,
		Capabilities: normalizeCapabilities(capabilities),
		NodeID:       nodeID,
		Status:       "offline",
		LastSeenAt:   nil,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	s.agents[agent.ID] = agent
	s.agentByNode[nodeID] = agent.ID
	if err := s.persistAgentUnsafe(agent); err != nil {
		return nil, mongoWriteError(err)
	}

	clone := copyAgent(agent)
	clone.Usage = s.agentUsageUnsafe(agent.ID)
	return clone, nil
}

func (s *Store) ListAgents(userID string) []model.Agent {
	s.mu.RLock()
	defer s.mu.RUnlock()

	items := make([]model.Agent, 0)
	for _, a := range s.agents {
		if a.UserID == userID {
			clone := copyAgent(a)
			clone.Usage = s.agentUsageUnsafe(a.ID)
			items = append(items, *clone)
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt.After(items[j].CreatedAt) })
	return items
}

func (s *Store) GetAgent(userID, agentID string) (*model.Agent, *transport.AppError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	a, ok := s.agents[agentID]
	if !ok || a.UserID != userID {
		return nil, transport.NotFound("agent not found")
	}
	clone := copyAgent(a)
	clone.Usage = s.agentUsageUnsafe(a.ID)
	return clone, nil
}

func (s *Store) UpdateAgent(userID, agentID string, in UpdateAgentInput) (*model.Agent, *transport.AppError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	a, ok := s.agents[agentID]
	if !ok || a.UserID != userID {
		return nil, transport.NotFound("agent not found")
	}

	if in.Name != nil {
		name := strings.TrimSpace(*in.Name)
		if name == "" {
			return nil, transport.Validation("invalid name", map[string]any{"name": "cannot be empty"})
		}
		a.Name = name
	}
	if in.Role != nil {
		role := strings.TrimSpace(*in.Role)
		if !isValidRole(role) {
			return nil, transport.Validation("invalid role", map[string]any{"role": "must be one of pm/developer/reviewer/custom"})
		}
		a.Role = role
	}
	if in.Description != nil {
		desc := strings.TrimSpace(*in.Description)
		if desc == "" {
			return nil, transport.Validation("invalid description", map[string]any{"description": "cannot be empty"})
		}
		a.Description = desc
	}
	if in.Capabilities != nil {
		a.Capabilities = normalizeCapabilities(*in.Capabilities)
	}
	a.UpdatedAt = time.Now().UTC()

	s.rebuildProjectPMSummariesUnsafe(a.ID)
	s.rebuildTaskPMSummariesUnsafe(a.ID)
	s.rebuildTodoAssigneeUnsafe(a.ID)
	if err := s.persistAgentGraphUnsafe(a.ID); err != nil {
		return nil, mongoWriteError(err)
	}

	clone := copyAgent(a)
	clone.Usage = s.agentUsageUnsafe(a.ID)
	return clone, nil
}

func (s *Store) DeleteAgent(userID, agentID string) *transport.AppError {
	s.mu.Lock()
	defer s.mu.Unlock()
	a, ok := s.agents[agentID]
	if !ok || a.UserID != userID {
		return transport.NotFound("agent not found")
	}

	usage := s.agentUsageUnsafe(agentID)
	if usage.InUse {
		err := transport.Conflict("AGENT_IN_USE", "agent is referenced by project or task")
		err.Details = map[string]any{
			"project_count": usage.ProjectCount,
			"task_count":    usage.TaskCount,
			"todo_count":    usage.TodoCount,
			"total_count":   usage.TotalCount,
		}
		return err
	}
	delete(s.agentByNode, a.NodeID)
	delete(s.agents, agentID)
	if err := s.deleteAgentUnsafe(agentID); err != nil {
		return mongoWriteError(err)
	}
	return nil
}

func (s *Store) agentByNodeUnsafe(nodeID string) (*model.Agent, *transport.AppError) {
	agentID, ok := s.agentByNode[nodeID]
	if !ok {
		return nil, transport.NotFound("agent not found by node_id")
	}
	agent, ok := s.agents[agentID]
	if !ok {
		return nil, transport.NotFound("agent not found")
	}
	return agent, nil
}

func (s *Store) agentUsageUnsafe(agentID string) model.AgentUsage {
	usage := model.AgentUsage{}
	for _, p := range s.projects {
		if p.PMAgentID == agentID {
			usage.ProjectCount++
		}
	}
	for _, t := range s.tasks {
		if t.PMAgentID == agentID {
			usage.TaskCount++
		}
		for _, todo := range t.Todos {
			if todo.Assignee.AgentID == agentID {
				usage.TodoCount++
			}
		}
	}
	usage.TotalCount = usage.ProjectCount + usage.TaskCount + usage.TodoCount
	usage.InUse = usage.TotalCount > 0
	return usage
}

func (s *Store) rebuildProjectPMSummariesUnsafe(agentID string) {
	a, ok := s.agents[agentID]
	if !ok {
		return
	}
	for _, p := range s.projects {
		if p.PMAgentID == agentID {
			p.PMAgent = toPMSummary(a)
		}
	}
}

func (s *Store) rebuildTaskPMSummariesUnsafe(agentID string) {
	a, ok := s.agents[agentID]
	if !ok {
		return
	}
	for _, t := range s.tasks {
		if t.PMAgentID == agentID {
			t.PMAgent = toPMSummary(a)
		}
	}
}

func (s *Store) rebuildTodoAssigneeUnsafe(agentID string) {
	a, ok := s.agents[agentID]
	if !ok {
		return
	}
	for _, t := range s.tasks {
		for i := range t.Todos {
			if t.Todos[i].Assignee.AgentID == agentID {
				t.Todos[i].Assignee.Name = a.Name
				t.Todos[i].Assignee.NodeID = a.NodeID
			}
		}
	}
}

func normalizeCapabilities(in []string) []string {
	if len(in) == 0 {
		return []string{}
	}
	out := make([]string, 0, len(in))
	seen := make(map[string]struct{}, len(in))
	for _, item := range in {
		v := strings.TrimSpace(item)
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}

func isValidRole(role string) bool {
	switch role {
	case "pm", "developer", "reviewer", "custom":
		return true
	default:
		return false
	}
}
