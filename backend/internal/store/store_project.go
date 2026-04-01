package store

import (
	"sort"
	"strings"
	"time"

	"trustmesh/backend/internal/model"
	"trustmesh/backend/internal/transport"
)

type UpdateProjectInput struct {
	Name        *string
	Description *string
	PMAgentID   *string
}

func (s *Store) buildProjectViewUnsafe(project *model.Project) *model.Project {
	clone := copyProject(project)
	clone.TaskSummary = s.aggregateProjectTaskSummaryUnsafe(project)
	return clone
}

func (s *Store) aggregateProjectTaskSummaryUnsafe(project *model.Project) model.ProjectTaskSummary {
	summary := model.ProjectTaskSummary{
		WorkStatus: "empty",
	}

	ids := s.projectTasks[project.ID]
	for _, id := range ids {
		task, ok := s.tasks[id]
		if !ok || task.UserID != project.UserID {
			continue
		}

		summary.TaskTotal++
		switch task.Status {
		case "pending":
			summary.PendingCount++
		case "in_progress":
			summary.InProgressCount++
		case "done":
			summary.DoneCount++
		case "failed":
			summary.FailedCount++
		case "canceled":
			summary.CanceledCount++
		}

		if summary.LatestTaskAt == nil || task.UpdatedAt.After(*summary.LatestTaskAt) {
			at := task.UpdatedAt
			summary.LatestTaskAt = &at
		}
	}

	switch {
	case project.Status == "archived":
		summary.WorkStatus = "archived"
	case summary.TaskTotal == 0:
		summary.WorkStatus = "empty"
	case summary.InProgressCount > 0:
		summary.WorkStatus = "running"
	case summary.FailedCount > 0:
		summary.WorkStatus = "attention"
	case summary.PendingCount > 0:
		summary.WorkStatus = "queued"
	default:
		summary.WorkStatus = "idle"
	}

	return summary
}

func (s *Store) CreateProject(userID, name, description, pmAgentID string) (*model.Project, *transport.AppError) {
	name = strings.TrimSpace(name)
	description = strings.TrimSpace(description)
	if name == "" || description == "" || strings.TrimSpace(pmAgentID) == "" {
		return nil, transport.Validation("invalid project payload", map[string]any{
			"name":        "required",
			"description": "required",
			"pm_agent_id": "required",
		})
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	pm, err := s.pmAgentForUserUnsafe(userID, pmAgentID)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	project := &model.Project{
		ID:          newID(),
		UserID:      userID,
		Name:        name,
		Description: description,
		Status:      "active",
		PMAgentID:   pmAgentID,
		PMAgent:     toPMSummary(pm),
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	s.projects[project.ID] = project
	if err := s.persistProjectUnsafe(project); err != nil {
		return nil, mongoWriteError(err)
	}
	return s.buildProjectViewUnsafe(project), nil
}

func (s *Store) ListProjects(userID string) []model.Project {
	s.mu.RLock()
	defer s.mu.RUnlock()

	items := make([]model.Project, 0)
	for _, p := range s.projects {
		if p.UserID == userID {
			items = append(items, *s.buildProjectViewUnsafe(p))
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt.After(items[j].CreatedAt) })
	return items
}

func (s *Store) GetProject(userID, projectID string) (*model.Project, *transport.AppError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.projects[projectID]
	if !ok || p.UserID != userID {
		return nil, transport.NotFound("project not found")
	}
	return s.buildProjectViewUnsafe(p), nil
}

func (s *Store) UpdateProject(userID, projectID string, in UpdateProjectInput) (*model.Project, *transport.AppError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.projects[projectID]
	if !ok || p.UserID != userID {
		return nil, transport.NotFound("project not found")
	}
	if in.Name != nil {
		name := strings.TrimSpace(*in.Name)
		if name == "" {
			return nil, transport.Validation("invalid name", map[string]any{"name": "cannot be empty"})
		}
		p.Name = name
	}
	if in.Description != nil {
		desc := strings.TrimSpace(*in.Description)
		if desc == "" {
			return nil, transport.Validation("invalid description", map[string]any{"description": "cannot be empty"})
		}
		p.Description = desc
	}
	if in.PMAgentID != nil {
		agentID := strings.TrimSpace(*in.PMAgentID)
		if agentID == "" {
			return nil, transport.Validation("invalid pm_agent_id", map[string]any{"pm_agent_id": "cannot be empty"})
		}
		pm, err := s.pmAgentForUserUnsafe(userID, agentID)
		if err != nil {
			return nil, err
		}
		p.PMAgentID = agentID
		p.PMAgent = toPMSummary(pm)
	}
	p.UpdatedAt = time.Now().UTC()
	if err := s.persistProjectUnsafe(p); err != nil {
		return nil, mongoWriteError(err)
	}
	return s.buildProjectViewUnsafe(p), nil
}

func (s *Store) ArchiveProject(userID, projectID string) (*model.Project, *transport.AppError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.projects[projectID]
	if !ok || p.UserID != userID {
		return nil, transport.NotFound("project not found")
	}

	now := time.Now().UTC()
	affectedAgents, err := s.resetArchivedProjectTasksUnsafe(p, now)
	if err != nil {
		return nil, mongoWriteError(err)
	}

	p.Status = "archived"
	p.UpdatedAt = now
	if err := s.persistProjectUnsafe(p); err != nil {
		return nil, mongoWriteError(err)
	}

	for agentID := range affectedAgents {
		s.refreshAgentExecutionStatusUnsafe(agentID, now)
		if err := s.persistAgentGraphUnsafe(agentID); err != nil {
			return nil, mongoWriteError(err)
		}
	}
	return s.buildProjectViewUnsafe(p), nil
}

func (s *Store) resetArchivedProjectTasksUnsafe(project *model.Project, now time.Time) (map[string]struct{}, error) {
	affectedAgents := make(map[string]struct{})

	for _, taskID := range s.projectTasks[project.ID] {
		task, ok := s.tasks[taskID]
		if !ok || task.UserID != project.UserID {
			continue
		}

		taskChanged := false
		for i := range task.Todos {
			todo := &task.Todos[i]
			if todo.Status != "in_progress" {
				continue
			}

			todo.Status = "pending"
			todo.StartedAt = nil
			todo.CompletedAt = nil
			todo.FailedAt = nil
			todo.Error = nil
			affectedAgents[todo.Assignee.AgentID] = struct{}{}
			taskChanged = true
		}

		if task.Status == "in_progress" {
			task.Status = "pending"
			taskChanged = true
		}

		if !taskChanged {
			continue
		}

		task.Result = aggregateTaskResult(task.Todos, task.Status)
		task.UpdatedAt = now
		task.Version++

		if err := s.persistTaskBundleUnsafe(task.ID); err != nil {
			return nil, err
		}
		s.publishTaskUnsafe(task.ID)
	}

	return affectedAgents, nil
}

func (s *Store) GetProjectPMNode(userID, projectID string) (string, *transport.AppError) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	project, err := s.projectForUserUnsafe(userID, projectID)
	if err != nil {
		return "", err
	}
	agent, ok := s.agents[project.PMAgentID]
	if !ok {
		return "", transport.Conflict("PROJECT_PM_AGENT_INVALID", "project bound PM agent is invalid")
	}
	return agent.NodeID, nil
}

func (s *Store) CheckTaskProjectActive(taskID string) *transport.AppError {
	s.mu.RLock()
	defer s.mu.RUnlock()

	task, ok := s.tasks[taskID]
	if !ok {
		return transport.NotFound("task not found")
	}
	return s.ensureTaskProjectActiveUnsafe(task)
}

func (s *Store) projectForUserUnsafe(userID, projectID string) (*model.Project, *transport.AppError) {
	p, ok := s.projects[projectID]
	if !ok || p.UserID != userID {
		return nil, transport.NotFound("project not found")
	}
	return p, nil
}

func (s *Store) ensureTaskProjectActiveUnsafe(task *model.TaskDetail) *transport.AppError {
	project, ok := s.projects[task.ProjectID]
	if !ok {
		return transport.NotFound("project not found")
	}
	if project.Status == "archived" {
		return transport.Conflict("PROJECT_ARCHIVED", "archived project cannot mutate tasks")
	}
	return nil
}

func (s *Store) pmAgentForUserUnsafe(userID, agentID string) (*model.Agent, *transport.AppError) {
	a, ok := s.agents[agentID]
	if !ok || a.UserID != userID || a.Role != "pm" || a.Archived {
		return nil, transport.Conflict("PROJECT_PM_AGENT_INVALID", "pm_agent_id must reference a PM agent of current user")
	}
	return a, nil
}

func (s *Store) validateProjectPMAgentOnlineUnsafe(project *model.Project) *transport.AppError {
	a, ok := s.agents[project.PMAgentID]
	if !ok || a.Role != "pm" {
		return transport.Conflict("PROJECT_PM_AGENT_INVALID", "project bound PM agent is invalid")
	}
	if a.Status == "offline" {
		return transport.Conflict("PM_AGENT_OFFLINE", "project pm agent is offline")
	}
	return nil
}
