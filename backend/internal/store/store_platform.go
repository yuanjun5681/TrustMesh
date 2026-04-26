package store

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"trustmesh/backend/internal/model"
	"trustmesh/backend/internal/transport"
)

// ── Platform connection CRUD ───────────────────────────────────────────────

type UpsertPlatformConnectionInput struct {
	Platform       string
	PlatformNodeID string
	RemoteUserID   string
	PMAgentID      string
}

func (s *Store) UpsertPlatformConnection(userID string, in UpsertPlatformConnectionInput) (*model.PlatformConnection, *transport.AppError) {
	in.Platform = strings.TrimSpace(in.Platform)
	in.PlatformNodeID = strings.TrimSpace(in.PlatformNodeID)
	in.RemoteUserID = strings.TrimSpace(in.RemoteUserID)
	in.PMAgentID = strings.TrimSpace(in.PMAgentID)

	if in.Platform == "" || in.PlatformNodeID == "" || in.RemoteUserID == "" || in.PMAgentID == "" {
		return nil, transport.Validation("invalid platform connection payload", map[string]any{
			"platform":         "required",
			"platform_node_id": "required",
			"remote_user_id":   "required",
			"pm_agent_id":      "required",
		})
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.users[userID]; !ok {
		return nil, transport.NotFound("user not found")
	}
	if _, err := s.pmAgentForUserUnsafe(userID, in.PMAgentID); err != nil {
		return nil, err
	}

	// Update existing binding for the same (user, platform, platformNodeID)
	for _, id := range s.userPlatformConns[userID] {
		c, ok := s.platformConns[id]
		if !ok {
			continue
		}
		if c.Platform == in.Platform && c.PlatformNodeID == in.PlatformNodeID {
			c.RemoteUserID = in.RemoteUserID
			c.PMAgentID = in.PMAgentID
			s.platformConnByNodeUser[platformConnKey(c.PlatformNodeID, c.RemoteUserID)] = c.ID

			if err := s.persistPlatformConnectionUnsafe(c); err != nil {
				return nil, mongoWriteError(err)
			}
			clone := *c
			return &clone, nil
		}
	}

	// New binding
	now := time.Now().UTC()
	conn := &model.PlatformConnection{
		ID:             newID(),
		UserID:         userID,
		Platform:       in.Platform,
		PlatformNodeID: in.PlatformNodeID,
		RemoteUserID:   in.RemoteUserID,
		PMAgentID:      in.PMAgentID,
		LinkedAt:       now,
	}
	s.platformConns[conn.ID] = conn
	s.userPlatformConns[userID] = append(s.userPlatformConns[userID], conn.ID)
	s.platformConnByNodeUser[platformConnKey(conn.PlatformNodeID, conn.RemoteUserID)] = conn.ID

	if err := s.persistPlatformConnectionUnsafe(conn); err != nil {
		return nil, mongoWriteError(err)
	}
	clone := *conn
	return &clone, nil
}

func (s *Store) ListPlatformConnections(userID string) []model.PlatformConnection {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ids := s.userPlatformConns[userID]
	result := make([]model.PlatformConnection, 0, len(ids))
	for _, id := range ids {
		if c, ok := s.platformConns[id]; ok {
			result = append(result, *c)
		}
	}
	return result
}

func (s *Store) DeletePlatformConnection(userID, platform, platformNodeID string) *transport.AppError {
	s.mu.Lock()
	defer s.mu.Unlock()

	ids := s.userPlatformConns[userID]
	for i, id := range ids {
		c, ok := s.platformConns[id]
		if !ok || c.Platform != platform || c.PlatformNodeID != platformNodeID {
			continue
		}
		delete(s.platformConns, id)
		delete(s.platformConnByNodeUser, platformConnKey(c.PlatformNodeID, c.RemoteUserID))
		s.userPlatformConns[userID] = append(ids[:i], ids[i+1:]...)

		if err := s.deletePlatformConnectionUnsafe(id); err != nil {
			return mongoWriteError(err)
		}
		return nil
	}
	return transport.NotFound("platform connection not found")
}

// LookupPlatformConnection finds the local binding for an inbound event.
// platformNodeID is the webhook.From value; remoteUserID comes from event metadata.
func (s *Store) LookupPlatformConnection(platformNodeID, remoteUserID string) (*model.PlatformConnection, *transport.AppError) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	id, ok := s.platformConnByNodeUser[platformConnKey(platformNodeID, remoteUserID)]
	if !ok {
		return nil, transport.NotFound("platform connection not found")
	}
	c, ok := s.platformConns[id]
	if !ok {
		return nil, transport.NotFound("platform connection not found")
	}
	clone := *c
	return &clone, nil
}

// ── ClawHire project ───────────────────────────────────────────────────────

// EnsureClawHireProject returns the project ID for the user's ClawHire binding,
// creating one (idempotently) if it does not yet exist.
func (s *Store) EnsureClawHireProject(userID string, conn *model.PlatformConnection) (string, *transport.AppError) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Look for an existing project scoped to this binding
	for _, p := range s.projects {
		if p.UserID == userID && p.SourcePlatformNodeID == conn.PlatformNodeID {
			return p.ID, nil
		}
	}

	pm, appErr := s.pmAgentForUserUnsafe(userID, conn.PMAgentID)
	if appErr != nil {
		return "", appErr
	}

	suffix := conn.PlatformNodeID
	if len(suffix) > 8 {
		suffix = suffix[len(suffix)-8:]
	}

	now := time.Now().UTC()
	project := &model.Project{
		ID:                   newID(),
		UserID:               userID,
		Name:                 fmt.Sprintf("ClawHire · %s", suffix),
		Description:          "ClawHire 平台承接的任务",
		Status:               "active",
		PMAgentID:            pm.ID,
		PMAgent:              toPMSummary(pm),
		SourcePlatform:       "clawhire",
		SourcePlatformNodeID: conn.PlatformNodeID,
		CreatedAt:            now,
		UpdatedAt:            now,
	}
	s.projects[project.ID] = project

	if err := s.persistProjectUnsafe(project); err != nil {
		return "", mongoWriteError(err)
	}
	return project.ID, nil
}

// ── External task creation ─────────────────────────────────────────────────

type ExternalTaskCreateInput struct {
	ProjectID   string
	Title       string
	Description string
	ExternalRef model.ExternalTaskRef
}

// CreateExternalTask creates a planning-phase task originated from an external platform.
// The returned task must be published to the PM agent by the caller.
func (s *Store) CreateExternalTask(userID string, in ExternalTaskCreateInput) (*model.TaskDetail, *transport.AppError) {
	in.Title = strings.TrimSpace(in.Title)
	in.Description = strings.TrimSpace(in.Description)
	if in.Title == "" || in.Description == "" || strings.TrimSpace(in.ProjectID) == "" {
		return nil, transport.Validation("invalid external task payload", map[string]any{
			"title":       "required",
			"description": "required",
			"project_id":  "required",
		})
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	project, ok := s.projects[in.ProjectID]
	if !ok || project.UserID != userID {
		return nil, transport.NotFound("project not found")
	}
	if project.Status == "archived" {
		return nil, transport.Conflict("PROJECT_ARCHIVED", "archived project cannot create tasks")
	}

	pmAgent, ok := s.agents[project.PMAgentID]
	if !ok {
		return nil, transport.Conflict("PROJECT_PM_AGENT_INVALID", "project bound PM agent is invalid")
	}

	now := time.Now().UTC()
	initialMsg := model.TaskMessage{
		ID:        uuid.NewString(),
		Role:      "user",
		Content:   in.Description,
		CreatedAt: now,
	}

	ref := in.ExternalRef
	task := &model.TaskDetail{
		ID:          newID(),
		UserID:      userID,
		ProjectID:   project.ID,
		Title:       in.Title,
		Description: in.Description,
		Status:      "planning",
		Priority:    "medium",
		PMAgentID:   pmAgent.ID,
		PMAgent:     toPMSummary(pmAgent),
		Messages:    []model.TaskMessage{initialMsg},
		Todos:       []model.Todo{},
		Result: model.TaskResult{
			Summary:     "",
			FinalOutput: "",
			Metadata:    map[string]any{},
		},
		ExternalRef: &ref,
		Version:     1,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	s.tasks[task.ID] = task
	s.projectTasks[task.ProjectID] = append(s.projectTasks[task.ProjectID], task.ID)

	taskTitle := task.Title
	s.addEventUnsafe(userID, project.ID, task.ID, "", "system", "", "clawhire", "task_created", &taskTitle, map[string]any{
		"task_title":       task.Title,
		"source_platform":  in.ExternalRef.Platform,
		"external_task_id": in.ExternalRef.ExternalTaskID,
	}, now)

	if err := s.persistTaskBundleUnsafe(task.ID); err != nil {
		return nil, mongoWriteError(err)
	}
	return copyTask(task), nil
}

// RecordExternalPlatformEvent appends a platform event to the task's event log.
func (s *Store) RecordExternalPlatformEvent(platform, externalTaskID, eventType, detail string) *transport.AppError {
	s.mu.Lock()
	defer s.mu.Unlock()

	var task *model.TaskDetail
	for _, t := range s.tasks {
		if t.ExternalRef != nil &&
			t.ExternalRef.Platform == platform &&
			t.ExternalRef.ExternalTaskID == externalTaskID {
			task = t
			break
		}
	}
	if task == nil {
		return transport.NotFound("task not found for external ref")
	}

	now := time.Now().UTC()
	s.addEventUnsafe(task.UserID, task.ProjectID, task.ID, "", "system", "", platform, eventType, &detail, map[string]any{
		"platform":         platform,
		"external_task_id": externalTaskID,
	}, now)

	if err := s.persistTaskEventsUnsafe(task.ID); err != nil {
		return mongoWriteError(err)
	}
	s.publishTaskUnsafe(task.ID)
	return nil
}

// GetTaskByExternalRef finds a task by its external platform reference.
func (s *Store) GetTaskByExternalRef(platform, externalTaskID string) (*model.TaskDetail, *transport.AppError) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, task := range s.tasks {
		if task.ExternalRef != nil &&
			task.ExternalRef.Platform == platform &&
			task.ExternalRef.ExternalTaskID == externalTaskID {
			return copyTask(task), nil
		}
	}
	return nil, transport.NotFound("task not found for external ref")
}
