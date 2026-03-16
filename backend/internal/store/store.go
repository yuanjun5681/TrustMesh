package store

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.uber.org/zap"
	"trustmesh/backend/internal/model"
	"trustmesh/backend/internal/transport"
)

type Store struct {
	mu sync.RWMutex

	users       map[string]*model.User
	usersByMail map[string]string

	agents      map[string]*model.Agent
	agentByNode map[string]string

	projects map[string]*model.Project

	conversations        map[string]*model.Conversation
	projectConversations map[string][]string

	tasks             map[string]*model.TaskDetail
	projectTasks      map[string][]string
	conversationTasks map[string]string
	taskEvents        map[string][]model.TaskEvent
	processedMessages map[string]processedMessage

	heartbeatTTL           time.Duration
	heartbeatSweepInterval time.Duration
	heartbeatStopCh        chan struct{}
	heartbeatDoneCh        chan struct{}

	mongoEnabled           bool
	mongoClient            *mongo.Client
	mongoUsers             *mongo.Collection
	mongoAgents            *mongo.Collection
	mongoProjects          *mongo.Collection
	mongoConversations     *mongo.Collection
	mongoTasks             *mongo.Collection
	mongoTaskEvents        *mongo.Collection
	mongoProcessedMessages *mongo.Collection
	mongoTimeout           time.Duration
	log                    *zap.Logger
}

type processedMessage struct {
	Action     string `bson:"action" json:"action"`
	ResourceID string `bson:"resource_id" json:"resource_id"`
}

func New() *Store {
	return &Store{
		users:                make(map[string]*model.User),
		usersByMail:          make(map[string]string),
		agents:               make(map[string]*model.Agent),
		agentByNode:          make(map[string]string),
		projects:             make(map[string]*model.Project),
		conversations:        make(map[string]*model.Conversation),
		projectConversations: make(map[string][]string),
		tasks:                make(map[string]*model.TaskDetail),
		projectTasks:         make(map[string][]string),
		conversationTasks:    make(map[string]string),
		taskEvents:           make(map[string][]model.TaskEvent),
		processedMessages:    make(map[string]processedMessage),
	}
}

func (s *Store) CreateUser(email, name, passwordHash string) (*model.User, *transport.AppError) {
	normalized := strings.ToLower(strings.TrimSpace(email))
	if normalized == "" || strings.TrimSpace(name) == "" || passwordHash == "" {
		return nil, transport.Validation("invalid register payload", map[string]any{"email": "required", "name": "required", "password": "required"})
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.usersByMail[normalized]; exists {
		return nil, transport.Conflict("EMAIL_EXISTS", "email already exists")
	}
	now := time.Now().UTC()
	id := newID()
	u := &model.User{
		ID:           id,
		Email:        normalized,
		Name:         strings.TrimSpace(name),
		PasswordHash: passwordHash,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	s.users[id] = u
	s.usersByMail[normalized] = id
	if err := s.persistUserUnsafe(u); err != nil {
		return nil, mongoWriteError(err)
	}

	return copyUser(u), nil
}

func (s *Store) FindUserByEmail(email string) (*model.User, bool) {
	normalized := strings.ToLower(strings.TrimSpace(email))
	s.mu.RLock()
	defer s.mu.RUnlock()
	id, ok := s.usersByMail[normalized]
	if !ok {
		return nil, false
	}
	u, ok := s.users[id]
	if !ok {
		return nil, false
	}
	return copyUser(u), true
}

func (s *Store) FindUserByID(userID string) (*model.User, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	u, ok := s.users[userID]
	if !ok {
		return nil, false
	}
	return copyUser(u), true
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
		Status:       "online",
		LastSeenAt:   &now,
		HeartbeatAt:  &now,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	s.agents[agent.ID] = agent
	s.agentByNode[nodeID] = agent.ID
	if err := s.persistAgentUnsafe(agent); err != nil {
		return nil, mongoWriteError(err)
	}

	return copyAgent(agent), nil
}

func (s *Store) ListAgents(userID string) []model.Agent {
	s.mu.RLock()
	defer s.mu.RUnlock()

	items := make([]model.Agent, 0)
	for _, a := range s.agents {
		if a.UserID == userID {
			items = append(items, *copyAgent(a))
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
	return copyAgent(a), nil
}

type UpdateAgentInput struct {
	Name         *string
	Role         *string
	Description  *string
	Capabilities *[]string
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

	return copyAgent(a), nil
}

func (s *Store) DeleteAgent(userID, agentID string) *transport.AppError {
	s.mu.Lock()
	defer s.mu.Unlock()
	a, ok := s.agents[agentID]
	if !ok || a.UserID != userID {
		return transport.NotFound("agent not found")
	}

	if s.agentInUseUnsafe(agentID) {
		return transport.Conflict("AGENT_IN_USE", "agent is referenced by project or task")
	}
	delete(s.agentByNode, a.NodeID)
	delete(s.agents, agentID)
	if err := s.deleteAgentUnsafe(agentID); err != nil {
		return mongoWriteError(err)
	}
	return nil
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
	return copyProject(project), nil
}

func (s *Store) ListProjects(userID string) []model.Project {
	s.mu.RLock()
	defer s.mu.RUnlock()

	items := make([]model.Project, 0)
	for _, p := range s.projects {
		if p.UserID == userID {
			items = append(items, *copyProject(p))
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
	return copyProject(p), nil
}

type UpdateProjectInput struct {
	Name        *string
	Description *string
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
	p.UpdatedAt = time.Now().UTC()
	if err := s.persistProjectUnsafe(p); err != nil {
		return nil, mongoWriteError(err)
	}
	return copyProject(p), nil
}

func (s *Store) ArchiveProject(userID, projectID string) (*model.Project, *transport.AppError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.projects[projectID]
	if !ok || p.UserID != userID {
		return nil, transport.NotFound("project not found")
	}
	p.Status = "archived"
	p.UpdatedAt = time.Now().UTC()
	if err := s.persistProjectUnsafe(p); err != nil {
		return nil, mongoWriteError(err)
	}
	return copyProject(p), nil
}

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
	detail := s.toConversationDetailUnsafe(conv)
	return &detail, nil
}

func (s *Store) ListTasks(userID, projectID, status string) ([]model.TaskListItem, *transport.AppError) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if _, err := s.projectForUserUnsafe(userID, projectID); err != nil {
		return nil, err
	}
	if status != "" && !isValidTaskStatus(status) {
		return nil, transport.Validation("invalid status", map[string]any{"status": "must be pending/in_progress/done/failed"})
	}

	ids := s.projectTasks[projectID]
	items := make([]model.TaskListItem, 0, len(ids))
	for _, id := range ids {
		task, ok := s.tasks[id]
		if !ok || task.UserID != userID {
			continue
		}
		if status != "" && task.Status != status {
			continue
		}
		items = append(items, toTaskListItem(*task))
	}
	sort.Slice(items, func(i, j int) bool { return items[i].UpdatedAt.After(items[j].UpdatedAt) })
	return items, nil
}

func (s *Store) GetTask(userID, taskID string) (*model.TaskDetail, *transport.AppError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	task, ok := s.tasks[taskID]
	if !ok || task.UserID != userID {
		return nil, transport.NotFound("task not found")
	}
	return copyTask(task), nil
}

func (s *Store) ListTaskEvents(userID, taskID string) ([]model.TaskEvent, *transport.AppError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	task, ok := s.tasks[taskID]
	if !ok || task.UserID != userID {
		return nil, transport.NotFound("task not found")
	}
	events := s.taskEvents[taskID]
	cloned := make([]model.TaskEvent, len(events))
	copy(cloned, events)
	sort.Slice(cloned, func(i, j int) bool { return cloned[i].CreatedAt.Before(cloned[j].CreatedAt) })
	return cloned, nil
}

func (s *Store) projectForUserUnsafe(userID, projectID string) (*model.Project, *transport.AppError) {
	p, ok := s.projects[projectID]
	if !ok || p.UserID != userID {
		return nil, transport.NotFound("project not found")
	}
	return p, nil
}

func (s *Store) pmAgentForUserUnsafe(userID, agentID string) (*model.Agent, *transport.AppError) {
	a, ok := s.agents[agentID]
	if !ok || a.UserID != userID || a.Role != "pm" {
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

func (s *Store) getTaskSummaryByConversationUnsafe(conversationID string) *model.TaskSummary {
	taskID, ok := s.conversationTasks[conversationID]
	if !ok {
		return nil
	}
	task, ok := s.tasks[taskID]
	if !ok {
		return nil
	}
	completed := 0
	for _, todo := range task.Todos {
		if todo.Status == "done" {
			completed++
		}
	}
	return &model.TaskSummary{
		ID:                 task.ID,
		Title:              task.Title,
		Status:             task.Status,
		Priority:           task.Priority,
		TodoCount:          len(task.Todos),
		CompletedTodoCount: completed,
		CreatedAt:          task.CreatedAt,
		UpdatedAt:          task.UpdatedAt,
	}
}

func (s *Store) agentInUseUnsafe(agentID string) bool {
	for _, p := range s.projects {
		if p.PMAgentID == agentID {
			return true
		}
	}
	for _, t := range s.tasks {
		if t.PMAgentID == agentID {
			return true
		}
		for _, todo := range t.Todos {
			if todo.Assignee.AgentID == agentID {
				return true
			}
		}
	}
	return false
}

func (s *Store) rebuildProjectPMSummariesUnsafe(agentID string) {
	a, ok := s.agents[agentID]
	if !ok {
		return
	}
	for _, p := range s.projects {
		if p.PMAgentID == agentID {
			p.PMAgent = toPMSummary(a)
			p.UpdatedAt = time.Now().UTC()
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
			t.UpdatedAt = time.Now().UTC()
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
				t.UpdatedAt = time.Now().UTC()
			}
		}
	}
}

func toTaskListItem(task model.TaskDetail) model.TaskListItem {
	completed := 0
	failed := 0
	for _, td := range task.Todos {
		if td.Status == "done" {
			completed++
		}
		if td.Status == "failed" {
			failed++
		}
	}
	return model.TaskListItem{
		ID:                 task.ID,
		ProjectID:          task.ProjectID,
		ConversationID:     task.ConversationID,
		Title:              task.Title,
		Description:        task.Description,
		Status:             task.Status,
		Priority:           task.Priority,
		PMAgent:            task.PMAgent,
		TodoCount:          len(task.Todos),
		CompletedTodoCount: completed,
		FailedTodoCount:    failed,
		CreatedAt:          task.CreatedAt,
		UpdatedAt:          task.UpdatedAt,
	}
}

func toPMSummary(a *model.Agent) model.PMAgentSummary {
	return model.PMAgentSummary{ID: a.ID, Name: a.Name, NodeID: a.NodeID, Status: a.Status}
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

func isValidTaskStatus(status string) bool {
	switch status {
	case "pending", "in_progress", "done", "failed":
		return true
	default:
		return false
	}
}

func newID() string {
	buf := make([]byte, 12)
	if _, err := rand.Read(buf); err != nil {
		panic(errors.New("failed to generate id"))
	}
	return hex.EncodeToString(buf)
}

func copyUser(u *model.User) *model.User {
	clone := *u
	return &clone
}

func copyAgent(a *model.Agent) *model.Agent {
	clone := *a
	if a.Capabilities != nil {
		clone.Capabilities = append([]string(nil), a.Capabilities...)
	}
	if a.LastSeenAt != nil {
		t := *a.LastSeenAt
		clone.LastSeenAt = &t
	}
	if a.HeartbeatAt != nil {
		t := *a.HeartbeatAt
		clone.HeartbeatAt = &t
	}
	return &clone
}

func copyProject(p *model.Project) *model.Project {
	clone := *p
	return &clone
}

func copyTask(t *model.TaskDetail) *model.TaskDetail {
	clone := *t
	clone.Todos = append([]model.Todo(nil), t.Todos...)
	clone.Artifacts = append([]model.TaskArtifact(nil), t.Artifacts...)
	clone.Result = model.TaskResult{
		Summary:     t.Result.Summary,
		FinalOutput: t.Result.FinalOutput,
		Metadata:    copyMap(t.Result.Metadata),
	}
	for i := range clone.Todos {
		clone.Todos[i].Result.Metadata = copyMap(clone.Todos[i].Result.Metadata)
		clone.Todos[i].Result.ArtifactRefs = append([]model.TodoResultArtifactRef(nil), clone.Todos[i].Result.ArtifactRefs...)
	}
	for i := range clone.Artifacts {
		clone.Artifacts[i].Metadata = copyMap(clone.Artifacts[i].Metadata)
	}
	return &clone
}

func copyMap(in map[string]any) map[string]any {
	if in == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func mongoWriteError(err error) *transport.AppError {
	return &transport.AppError{
		Status:  500,
		Code:    "INTERNAL_ERROR",
		Message: "failed to persist state",
		Details: map[string]any{"cause": err.Error()},
	}
}
