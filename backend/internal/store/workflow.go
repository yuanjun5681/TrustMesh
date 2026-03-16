package store

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"trustmesh/backend/internal/model"
	"trustmesh/backend/internal/transport"
)

type TaskCreateTodoInput struct {
	ID             string
	Title          string
	Description    string
	AssigneeNodeID string
}

type TaskCreateInput struct {
	ProjectID      string
	ConversationID string
	Title          string
	Description    string
	Todos          []TaskCreateTodoInput
}

type TodoProgressInput struct {
	TaskID  string
	TodoID  string
	Message string
}

type TodoCompleteInput struct {
	TaskID string
	TodoID string
	Result model.TodoResult
}

type TodoFailInput struct {
	TaskID string
	TodoID string
	Error  string
}

type AssignedTodoItem struct {
	TaskID         string     `json:"task_id"`
	ProjectID      string     `json:"project_id"`
	ConversationID string     `json:"conversation_id"`
	TaskTitle      string     `json:"task_title"`
	TaskStatus     string     `json:"task_status"`
	Todo           model.Todo `json:"todo"`
}

type ProjectSummary struct {
	ID                        string               `json:"id"`
	Name                      string               `json:"name"`
	Description               string               `json:"description"`
	Status                    string               `json:"status"`
	PMAgent                   model.PMAgentSummary `json:"pm_agent"`
	ActiveConversationCount   int                  `json:"active_conversation_count"`
	ResolvedConversationCount int                  `json:"resolved_conversation_count"`
	TaskCount                 int                  `json:"task_count"`
	PendingTaskCount          int                  `json:"pending_task_count"`
	InProgressTaskCount       int                  `json:"in_progress_task_count"`
	DoneTaskCount             int                  `json:"done_task_count"`
	FailedTaskCount           int                  `json:"failed_task_count"`
	CreatedAt                 time.Time            `json:"created_at"`
	UpdatedAt                 time.Time            `json:"updated_at"`
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

func (s *Store) GetProjectSummaryForPMNode(nodeID, projectID string) (*ProjectSummary, *transport.AppError) {
	nodeID = strings.TrimSpace(nodeID)
	projectID = strings.TrimSpace(projectID)
	if nodeID == "" || projectID == "" {
		return nil, transport.Validation("invalid rpc payload", map[string]any{"node_id": "required", "project_id": "required"})
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	pmAgent, project, err := s.requirePMOnProjectUnsafe(nodeID, projectID)
	if err != nil {
		return nil, err
	}

	var activeConversationCount int
	var resolvedConversationCount int
	for _, conversationID := range s.projectConversations[project.ID] {
		conversation, ok := s.conversations[conversationID]
		if !ok {
			continue
		}
		switch conversation.Status {
		case "active":
			activeConversationCount++
		case "resolved":
			resolvedConversationCount++
		}
	}

	var pendingTaskCount int
	var inProgressTaskCount int
	var doneTaskCount int
	var failedTaskCount int
	for _, taskID := range s.projectTasks[project.ID] {
		task, ok := s.tasks[taskID]
		if !ok {
			continue
		}
		switch task.Status {
		case "pending":
			pendingTaskCount++
		case "in_progress":
			inProgressTaskCount++
		case "done":
			doneTaskCount++
		case "failed":
			failedTaskCount++
		}
	}

	return &ProjectSummary{
		ID:                        project.ID,
		Name:                      project.Name,
		Description:               project.Description,
		Status:                    project.Status,
		PMAgent:                   toPMSummary(pmAgent),
		ActiveConversationCount:   activeConversationCount,
		ResolvedConversationCount: resolvedConversationCount,
		TaskCount:                 len(s.projectTasks[project.ID]),
		PendingTaskCount:          pendingTaskCount,
		InProgressTaskCount:       inProgressTaskCount,
		DoneTaskCount:             doneTaskCount,
		FailedTaskCount:           failedTaskCount,
		CreatedAt:                 project.CreatedAt,
		UpdatedAt:                 project.UpdatedAt,
	}, nil
}

func (s *Store) UpdateAgentHeartbeat(nodeID, status string, ts time.Time) (*model.Agent, *transport.AppError) {
	nodeID = strings.TrimSpace(nodeID)
	if nodeID == "" {
		return nil, transport.Validation("invalid node_id", map[string]any{"node_id": "required"})
	}
	if status != "online" && status != "busy" {
		return nil, transport.Validation("invalid heartbeat status", map[string]any{"status": "must be online or busy"})
	}
	if ts.IsZero() {
		ts = time.Now().UTC()
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	agent, err := s.agentByNodeUnsafe(nodeID)
	if err != nil {
		return nil, err
	}
	agent.Status = status
	agent.HeartbeatAt = &ts
	agent.LastSeenAt = &ts
	agent.UpdatedAt = time.Now().UTC()
	s.rebuildProjectPMSummariesUnsafe(agent.ID)
	s.rebuildTaskPMSummariesUnsafe(agent.ID)
	if err := s.persistAgentGraphUnsafe(agent.ID); err != nil {
		return nil, mongoWriteError(err)
	}
	return copyAgent(agent), nil
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
	conv.Messages = append(conv.Messages, model.ConversationMessage{ID: uuid.NewString(), Role: "pm_agent", Content: content, CreatedAt: now})
	conv.UpdatedAt = now
	if err := s.persistConversationUnsafe(conv); err != nil {
		return nil, mongoWriteError(err)
	}
	detail := s.toConversationDetailUnsafe(conv)
	return &detail, nil
}

func (s *Store) CreateTaskByPMNode(nodeID string, in TaskCreateInput) (*model.TaskDetail, *transport.AppError) {
	return s.CreateTaskByPMNodeWithMessageID(nodeID, "", in)
}

func (s *Store) CreateTaskByPMNodeWithMessageID(nodeID, messageID string, in TaskCreateInput) (*model.TaskDetail, *transport.AppError) {
	in.Title = strings.TrimSpace(in.Title)
	in.Description = strings.TrimSpace(in.Description)
	if strings.TrimSpace(in.ProjectID) == "" || strings.TrimSpace(in.ConversationID) == "" || in.Title == "" || in.Description == "" {
		return nil, transport.Validation("invalid task.create payload", map[string]any{
			"project_id":      "required",
			"conversation_id": "required",
			"title":           "required",
			"description":     "required",
		})
	}
	if len(in.Todos) == 0 {
		return nil, transport.Validation("invalid task.create payload", map[string]any{"todos": "must not be empty"})
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if task, ok := s.findProcessedTaskUnsafe(processedMessageKey("task.create", nodeID, messageID)); ok {
		return task, nil
	}

	pmAgent, err := s.agentByNodeUnsafe(nodeID)
	if err != nil {
		return nil, err
	}
	if pmAgent.Role != "pm" {
		return nil, transport.Forbidden("only pm agent can create tasks")
	}

	project, ok := s.projects[in.ProjectID]
	if !ok {
		return nil, transport.NotFound("project not found")
	}
	if project.PMAgentID != pmAgent.ID {
		return nil, transport.Forbidden("pm agent is not bound to this project")
	}

	conv, ok := s.conversations[in.ConversationID]
	if !ok {
		return nil, transport.NotFound("conversation not found")
	}
	if conv.ProjectID != project.ID {
		return nil, transport.Validation("conversation and project mismatch", map[string]any{"conversation_id": "does not belong to project_id"})
	}
	if _, exists := s.conversationTasks[in.ConversationID]; exists {
		return nil, transport.Conflict("CONVERSATION_TASK_EXISTS", "conversation already linked to task")
	}
	if conv.Status != "active" {
		return nil, transport.Conflict("CONVERSATION_RESOLVED", "conversation is resolved")
	}

	now := time.Now().UTC()
	seenTodoIDs := make(map[string]struct{}, len(in.Todos))
	todos := make([]model.Todo, 0, len(in.Todos))
	for i, todoIn := range in.Todos {
		title := strings.TrimSpace(todoIn.Title)
		desc := strings.TrimSpace(todoIn.Description)
		assigneeNode := strings.TrimSpace(todoIn.AssigneeNodeID)
		if title == "" || desc == "" || assigneeNode == "" {
			return nil, transport.Validation("invalid todo in task.create", map[string]any{"todo_index": i, "title": "required", "description": "required", "assignee_node_id": "required"})
		}

		assigneeAgent, assigneeErr := s.agentByNodeUnsafe(assigneeNode)
		if assigneeErr != nil {
			return nil, transport.Validation("invalid assignee_node_id", map[string]any{"todo_index": i, "assignee_node_id": assigneeNode})
		}
		if assigneeAgent.UserID != project.UserID {
			return nil, transport.Forbidden("assignee agent does not belong to same user")
		}

		todoID := strings.TrimSpace(todoIn.ID)
		if todoID == "" {
			todoID = uuid.NewString()
		}
		if _, dup := seenTodoIDs[todoID]; dup {
			return nil, transport.Validation("duplicated todo id", map[string]any{"todo_id": todoID})
		}
		seenTodoIDs[todoID] = struct{}{}

		todos = append(todos, model.Todo{
			ID:          todoID,
			Title:       title,
			Description: desc,
			Status:      "pending",
			Assignee: model.TodoAssignee{
				AgentID: assigneeAgent.ID,
				Name:    assigneeAgent.Name,
				NodeID:  assigneeAgent.NodeID,
			},
			StartedAt:   nil,
			CompletedAt: nil,
			FailedAt:    nil,
			Error:       nil,
			Result: model.TodoResult{
				Summary:      "",
				Output:       "",
				ArtifactRefs: []model.TodoResultArtifactRef{},
				Metadata:     map[string]any{},
			},
			CreatedAt: now,
		})
	}

	task := &model.TaskDetail{
		ID:             newID(),
		UserID:         project.UserID,
		ProjectID:      project.ID,
		ConversationID: in.ConversationID,
		Title:          in.Title,
		Description:    in.Description,
		Status:         "pending",
		Priority:       "medium",
		PMAgentID:      pmAgent.ID,
		PMAgent:        toPMSummary(pmAgent),
		Todos:          todos,
		Artifacts:      []model.TaskArtifact{},
		Result: model.TaskResult{
			Summary:     "",
			FinalOutput: "",
			Metadata:    map[string]any{},
		},
		Version:   1,
		CreatedAt: now,
		UpdatedAt: now,
	}

	s.tasks[task.ID] = task
	s.projectTasks[task.ProjectID] = append(s.projectTasks[task.ProjectID], task.ID)
	s.conversationTasks[task.ConversationID] = task.ID

	conv.Status = "resolved"
	conv.UpdatedAt = now

	taskTitle := task.Title
	s.addTaskEventUnsafe(task.ID, "agent", pmAgent.ID, "task_created", &taskTitle, map[string]any{"conversation_id": task.ConversationID}, now)
	for _, todo := range task.Todos {
		title := fmt.Sprintf("assigned todo: %s", todo.Title)
		s.addTaskEventUnsafe(task.ID, "agent", pmAgent.ID, "todo_assigned", &title, map[string]any{"todo_id": todo.ID, "assignee_agent_id": todo.Assignee.AgentID}, now)
	}

	s.rememberProcessedMessageUnsafe(processedMessageKey("task.create", nodeID, messageID), "task.create", task.ID)
	if err := s.persistConversationUnsafe(conv); err != nil {
		return nil, mongoWriteError(err)
	}
	if err := s.persistTaskBundleUnsafe(task.ID); err != nil {
		return nil, mongoWriteError(err)
	}
	if err := s.persistProcessedMessageUnsafe(processedMessageKey("task.create", nodeID, messageID)); err != nil {
		return nil, mongoWriteError(err)
	}

	return copyTask(task), nil
}

func (s *Store) UpdateTodoProgressByNode(nodeID string, in TodoProgressInput) (*model.TaskDetail, *transport.AppError) {
	in.TaskID = strings.TrimSpace(in.TaskID)
	in.TodoID = strings.TrimSpace(in.TodoID)
	in.Message = strings.TrimSpace(in.Message)
	if in.TaskID == "" || in.TodoID == "" || in.Message == "" {
		return nil, transport.Validation("invalid todo.progress payload", map[string]any{"task_id": "required", "todo_id": "required", "message": "required"})
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	agent, err := s.agentByNodeUnsafe(nodeID)
	if err != nil {
		return nil, err
	}
	task, ok := s.tasks[in.TaskID]
	if !ok || task.UserID != agent.UserID {
		return nil, transport.NotFound("task not found")
	}

	todoIdx := findTodoIndex(task, in.TodoID)
	if todoIdx < 0 {
		return nil, transport.NotFound("todo not found")
	}
	todo := &task.Todos[todoIdx]
	if todo.Assignee.AgentID != agent.ID {
		return nil, transport.Forbidden("todo is not assigned to this agent")
	}
	if todo.Status == "done" || todo.Status == "failed" {
		return nil, transport.Conflict("TODO_FINALIZED", "todo already finalized")
	}

	now := time.Now().UTC()
	if todo.Status == "pending" {
		todo.Status = "in_progress"
		todo.StartedAt = &now
		started := fmt.Sprintf("todo started: %s", todo.Title)
		s.addTaskEventUnsafe(task.ID, "agent", agent.ID, "todo_started", &started, map[string]any{"todo_id": todo.ID}, now)
	}
	progress := in.Message
	s.addTaskEventUnsafe(task.ID, "agent", agent.ID, "todo_progress", &progress, map[string]any{"todo_id": todo.ID}, now)

	s.updateTaskStatusUnsafe(task, now)
	if err := s.persistTaskBundleUnsafe(task.ID); err != nil {
		return nil, mongoWriteError(err)
	}
	return copyTask(task), nil
}

func (s *Store) CompleteTodoByNode(nodeID string, in TodoCompleteInput) (*model.TaskDetail, *transport.AppError) {
	return s.CompleteTodoByNodeWithMessageID(nodeID, "", in)
}

func (s *Store) CompleteTodoByNodeWithMessageID(nodeID, messageID string, in TodoCompleteInput) (*model.TaskDetail, *transport.AppError) {
	in.TaskID = strings.TrimSpace(in.TaskID)
	in.TodoID = strings.TrimSpace(in.TodoID)
	if in.TaskID == "" || in.TodoID == "" {
		return nil, transport.Validation("invalid todo.complete payload", map[string]any{"task_id": "required", "todo_id": "required"})
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if task, ok := s.findProcessedTaskUnsafe(processedMessageKey("todo.complete", nodeID, messageID)); ok {
		return task, nil
	}

	agent, err := s.agentByNodeUnsafe(nodeID)
	if err != nil {
		return nil, err
	}
	task, ok := s.tasks[in.TaskID]
	if !ok || task.UserID != agent.UserID {
		return nil, transport.NotFound("task not found")
	}
	todoIdx := findTodoIndex(task, in.TodoID)
	if todoIdx < 0 {
		return nil, transport.NotFound("todo not found")
	}
	todo := &task.Todos[todoIdx]
	if todo.Assignee.AgentID != agent.ID {
		return nil, transport.Forbidden("todo is not assigned to this agent")
	}
	if todo.Status == "done" {
		return nil, transport.Conflict("TODO_ALREADY_DONE", "todo already done")
	}
	if todo.Status == "failed" {
		return nil, transport.Conflict("TODO_ALREADY_FAILED", "todo already failed")
	}

	now := time.Now().UTC()
	if todo.StartedAt == nil {
		todo.StartedAt = &now
	}
	todo.Status = "done"
	todo.CompletedAt = &now
	todo.FailedAt = nil
	todo.Error = nil
	todo.Result = model.TodoResult{
		Summary:      strings.TrimSpace(in.Result.Summary),
		Output:       strings.TrimSpace(in.Result.Output),
		ArtifactRefs: append([]model.TodoResultArtifactRef(nil), in.Result.ArtifactRefs...),
		Metadata:     copyMap(in.Result.Metadata),
	}
	completed := fmt.Sprintf("todo completed: %s", todo.Title)
	s.addTaskEventUnsafe(task.ID, "agent", agent.ID, "todo_completed", &completed, map[string]any{"todo_id": todo.ID}, now)

	s.updateTaskStatusUnsafe(task, now)
	s.rememberProcessedMessageUnsafe(processedMessageKey("todo.complete", nodeID, messageID), "todo.complete", task.ID)
	if err := s.persistTaskBundleUnsafe(task.ID); err != nil {
		return nil, mongoWriteError(err)
	}
	if err := s.persistProcessedMessageUnsafe(processedMessageKey("todo.complete", nodeID, messageID)); err != nil {
		return nil, mongoWriteError(err)
	}
	return copyTask(task), nil
}

func (s *Store) FailTodoByNode(nodeID string, in TodoFailInput) (*model.TaskDetail, *transport.AppError) {
	in.TaskID = strings.TrimSpace(in.TaskID)
	in.TodoID = strings.TrimSpace(in.TodoID)
	in.Error = strings.TrimSpace(in.Error)
	if in.TaskID == "" || in.TodoID == "" || in.Error == "" {
		return nil, transport.Validation("invalid todo.fail payload", map[string]any{"task_id": "required", "todo_id": "required", "error": "required"})
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	agent, err := s.agentByNodeUnsafe(nodeID)
	if err != nil {
		return nil, err
	}
	task, ok := s.tasks[in.TaskID]
	if !ok || task.UserID != agent.UserID {
		return nil, transport.NotFound("task not found")
	}
	todoIdx := findTodoIndex(task, in.TodoID)
	if todoIdx < 0 {
		return nil, transport.NotFound("todo not found")
	}
	todo := &task.Todos[todoIdx]
	if todo.Assignee.AgentID != agent.ID {
		return nil, transport.Forbidden("todo is not assigned to this agent")
	}
	if todo.Status == "done" {
		return nil, transport.Conflict("TODO_ALREADY_DONE", "todo already done")
	}
	if todo.Status == "failed" {
		return nil, transport.Conflict("TODO_ALREADY_FAILED", "todo already failed")
	}

	now := time.Now().UTC()
	if todo.StartedAt == nil {
		todo.StartedAt = &now
	}
	todo.Status = "failed"
	todo.CompletedAt = nil
	todo.FailedAt = &now
	errCopy := in.Error
	todo.Error = &errCopy
	failed := fmt.Sprintf("todo failed: %s", todo.Title)
	s.addTaskEventUnsafe(task.ID, "agent", agent.ID, "todo_failed", &failed, map[string]any{"todo_id": todo.ID, "error": in.Error}, now)

	s.updateTaskStatusUnsafe(task, now)
	if err := s.persistTaskBundleUnsafe(task.ID); err != nil {
		return nil, mongoWriteError(err)
	}
	return copyTask(task), nil
}

func (s *Store) GetTaskForNode(nodeID, taskID string) (*model.TaskDetail, *transport.AppError) {
	nodeID = strings.TrimSpace(nodeID)
	taskID = strings.TrimSpace(taskID)
	if nodeID == "" || taskID == "" {
		return nil, transport.Validation("invalid rpc payload", map[string]any{"node_id": "required", "task_id": "required"})
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	agent, err := s.agentByNodeUnsafe(nodeID)
	if err != nil {
		return nil, err
	}
	task, ok := s.tasks[taskID]
	if !ok || task.UserID != agent.UserID {
		return nil, transport.NotFound("task not found")
	}
	if !canAccessTaskUnsafe(agent.ID, task) {
		return nil, transport.Forbidden("agent has no access to this task")
	}
	return copyTask(task), nil
}

func (s *Store) GetTaskByConversationForPMNode(nodeID, conversationID string) (*model.TaskDetail, *transport.AppError) {
	nodeID = strings.TrimSpace(nodeID)
	conversationID = strings.TrimSpace(conversationID)
	if nodeID == "" || conversationID == "" {
		return nil, transport.Validation("invalid rpc payload", map[string]any{"node_id": "required", "conversation_id": "required"})
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	pmAgent, err := s.requirePMAgentUnsafe(nodeID)
	if err != nil {
		return nil, err
	}

	conversation, ok := s.conversations[conversationID]
	if !ok {
		return nil, transport.NotFound("conversation not found")
	}
	project, ok := s.projects[conversation.ProjectID]
	if !ok {
		return nil, transport.NotFound("project not found")
	}
	if project.PMAgentID != pmAgent.ID {
		return nil, transport.Forbidden("pm agent is not bound to this project")
	}

	taskID, ok := s.conversationTasks[conversationID]
	if !ok {
		return nil, nil
	}
	task, ok := s.tasks[taskID]
	if !ok {
		return nil, nil
	}
	return copyTask(task), nil
}

func (s *Store) ListAssignedTodosForNode(nodeID string) ([]AssignedTodoItem, *transport.AppError) {
	nodeID = strings.TrimSpace(nodeID)
	if nodeID == "" {
		return nil, transport.Validation("invalid node_id", map[string]any{"node_id": "required"})
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	agent, err := s.agentByNodeUnsafe(nodeID)
	if err != nil {
		return nil, err
	}

	items := make([]AssignedTodoItem, 0)
	for _, task := range s.tasks {
		if task.UserID != agent.UserID {
			continue
		}
		for _, todo := range task.Todos {
			if todo.Assignee.AgentID != agent.ID {
				continue
			}
			if todo.Status == "done" || todo.Status == "failed" {
				continue
			}
			items = append(items, AssignedTodoItem{
				TaskID:         task.ID,
				ProjectID:      task.ProjectID,
				ConversationID: task.ConversationID,
				TaskTitle:      task.Title,
				TaskStatus:     task.Status,
				Todo:           copyTodo(todo),
			})
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Todo.CreatedAt.Before(items[j].Todo.CreatedAt) })
	return items, nil
}

func (s *Store) ListCandidateAgentsForPMNode(nodeID, projectID string) ([]model.Agent, *transport.AppError) {
	nodeID = strings.TrimSpace(nodeID)
	projectID = strings.TrimSpace(projectID)
	if nodeID == "" {
		return nil, transport.Validation("invalid rpc payload", map[string]any{"node_id": "required"})
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	pmAgent, err := s.requirePMAgentUnsafe(nodeID)
	if err != nil {
		return nil, err
	}
	if projectID != "" {
		project, ok := s.projects[projectID]
		if !ok {
			return nil, transport.NotFound("project not found")
		}
		if project.PMAgentID != pmAgent.ID {
			return nil, transport.Forbidden("pm agent is not bound to this project")
		}
	}

	items := make([]model.Agent, 0)
	for _, agent := range s.agents {
		if agent.UserID != pmAgent.UserID {
			continue
		}
		items = append(items, *copyAgent(agent))
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Status != items[j].Status {
			return items[i].Status < items[j].Status
		}
		return items[i].Name < items[j].Name
	})
	return items, nil
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

func (s *Store) requirePMAgentUnsafe(nodeID string) (*model.Agent, *transport.AppError) {
	agent, err := s.agentByNodeUnsafe(nodeID)
	if err != nil {
		return nil, err
	}
	if agent.Role != "pm" {
		return nil, transport.Forbidden("only pm agent can access this rpc")
	}
	return agent, nil
}

func (s *Store) requirePMOnProjectUnsafe(nodeID, projectID string) (*model.Agent, *model.Project, *transport.AppError) {
	pmAgent, err := s.requirePMAgentUnsafe(nodeID)
	if err != nil {
		return nil, nil, err
	}
	project, ok := s.projects[projectID]
	if !ok {
		return nil, nil, transport.NotFound("project not found")
	}
	if project.PMAgentID != pmAgent.ID {
		return nil, nil, transport.Forbidden("pm agent is not bound to this project")
	}
	return pmAgent, project, nil
}

func processedMessageKey(action, nodeID, messageID string) string {
	if strings.TrimSpace(messageID) == "" {
		return ""
	}
	return action + "|" + nodeID + "|" + messageID
}

func (s *Store) findProcessedTaskUnsafe(key string) (*model.TaskDetail, bool) {
	if key == "" {
		return nil, false
	}
	record, ok := s.processedMessages[key]
	if !ok {
		return nil, false
	}
	task, ok := s.tasks[record.ResourceID]
	if !ok {
		return nil, false
	}
	return copyTask(task), true
}

func (s *Store) rememberProcessedMessageUnsafe(key, action, resourceID string) {
	if key == "" {
		return
	}
	s.processedMessages[key] = processedMessage{
		Action:     action,
		ResourceID: resourceID,
	}
}

func (s *Store) addTaskEventUnsafe(taskID, actorType, actorID, eventType string, content *string, metadata map[string]any, at time.Time) {
	event := model.TaskEvent{
		ID:        newID(),
		TaskID:    taskID,
		ActorType: actorType,
		ActorID:   actorID,
		EventType: eventType,
		Content:   content,
		Metadata:  copyMap(metadata),
		CreatedAt: at,
	}
	s.taskEvents[taskID] = append(s.taskEvents[taskID], event)
}

func (s *Store) updateTaskStatusUnsafe(task *model.TaskDetail, now time.Time) {
	prev := task.Status
	next := aggregateTaskStatus(*task)
	artifacts := aggregateTaskArtifacts(task.Todos)
	result := aggregateTaskResult(task.Todos, next, artifacts)
	task.Status = next
	task.Artifacts = artifacts
	task.Result = result
	task.UpdatedAt = now
	task.Version++
	if prev != next {
		msg := fmt.Sprintf("task status changed: %s -> %s", prev, next)
		s.addTaskEventUnsafe(task.ID, "system", "system", "task_status_changed", &msg, map[string]any{"from": prev, "to": next}, now)
	}
}

func aggregateTaskStatus(task model.TaskDetail) string {
	if len(task.Todos) == 0 {
		return "pending"
	}
	allDone := true
	hasWork := false
	for _, todo := range task.Todos {
		switch todo.Status {
		case "failed":
			return "failed"
		case "done":
			hasWork = true
		case "in_progress":
			allDone = false
			hasWork = true
		default:
			allDone = false
		}
	}
	if allDone {
		return "done"
	}
	if hasWork {
		return "in_progress"
	}
	return "pending"
}

func aggregateTaskArtifacts(todos []model.Todo) []model.TaskArtifact {
	artifacts := make([]model.TaskArtifact, 0)
	usedIDs := make(map[string]int)
	for _, todo := range todos {
		if todo.Status != "done" {
			continue
		}
		for i, ref := range todo.Result.ArtifactRefs {
			baseID := strings.TrimSpace(ref.ArtifactID)
			if baseID == "" {
				baseID = fmt.Sprintf("%s-artifact-%d", todo.ID, i+1)
			}
			artifactID := uniqueTaskArtifactID(baseID, usedIDs)
			sourceTodoID := todo.ID
			artifacts = append(artifacts, model.TaskArtifact{
				ID:           artifactID,
				SourceTodoID: &sourceTodoID,
				Kind:         strings.TrimSpace(ref.Kind),
				Title:        strings.TrimSpace(ref.Label),
				URI:          "",
				MimeType:     nil,
				Metadata: map[string]any{
					"source":   "todo_result_ref",
					"ref_only": true,
				},
			})
		}
	}
	return artifacts
}

func aggregateTaskResult(todos []model.Todo, status string, artifacts []model.TaskArtifact) model.TaskResult {
	total := len(todos)
	doneCount := 0
	failedCount := 0
	inProgressCount := 0
	pendingCount := 0
	completedSummaries := make([]string, 0)
	completedOutputs := make([]string, 0)
	failedMessages := make([]string, 0)

	for _, todo := range todos {
		switch todo.Status {
		case "done":
			doneCount++
			if summary := strings.TrimSpace(todo.Result.Summary); summary != "" {
				completedSummaries = append(completedSummaries, fmt.Sprintf("%s: %s", todo.Title, summary))
			}
			if output := strings.TrimSpace(todo.Result.Output); output != "" {
				completedOutputs = append(completedOutputs, fmt.Sprintf("%s\n%s", todo.Title, output))
			}
		case "failed":
			failedCount++
			if todo.Error != nil && strings.TrimSpace(*todo.Error) != "" {
				failedMessages = append(failedMessages, fmt.Sprintf("%s: %s", todo.Title, strings.TrimSpace(*todo.Error)))
			} else {
				failedMessages = append(failedMessages, fmt.Sprintf("%s: failed", todo.Title))
			}
		case "in_progress":
			inProgressCount++
		default:
			pendingCount++
		}
	}

	summary := ""
	finalOutput := ""
	switch status {
	case "done":
		if len(completedSummaries) > 0 {
			summary = strings.Join(completedSummaries, "; ")
		} else if total > 0 {
			summary = fmt.Sprintf("All %d todos completed", total)
		}
		if len(completedOutputs) > 0 {
			finalOutput = strings.Join(completedOutputs, "\n\n")
		}
	case "failed":
		if len(failedMessages) > 0 {
			summary = "Task failed: " + strings.Join(failedMessages, "; ")
		} else {
			summary = "Task failed"
		}
		sections := make([]string, 0, 2)
		if len(completedOutputs) > 0 {
			sections = append(sections, strings.Join(completedOutputs, "\n\n"))
		}
		if len(failedMessages) > 0 {
			sections = append(sections, "Failed todos:\n"+strings.Join(failedMessages, "\n"))
		}
		finalOutput = strings.Join(sections, "\n\n")
	case "in_progress":
		summary = fmt.Sprintf("Task in progress: %d/%d completed, %d in progress, %d pending", doneCount, total, inProgressCount, pendingCount)
	case "pending":
		summary = fmt.Sprintf("Task pending: %d todos not started", total)
	}

	return model.TaskResult{
		Summary:     summary,
		FinalOutput: finalOutput,
		Metadata: map[string]any{
			"status":                 status,
			"todo_count":             total,
			"completed_todo_count":   doneCount,
			"failed_todo_count":      failedCount,
			"in_progress_todo_count": inProgressCount,
			"pending_todo_count":     pendingCount,
			"artifact_count":         len(artifacts),
		},
	}
}

func uniqueTaskArtifactID(baseID string, usedIDs map[string]int) string {
	count := usedIDs[baseID]
	usedIDs[baseID] = count + 1
	if count == 0 {
		return baseID
	}
	return fmt.Sprintf("%s-%d", baseID, count+1)
}

func findTodoIndex(task *model.TaskDetail, todoID string) int {
	for i := range task.Todos {
		if task.Todos[i].ID == todoID {
			return i
		}
	}
	return -1
}

func canAccessTaskUnsafe(agentID string, task *model.TaskDetail) bool {
	if task.PMAgentID == agentID {
		return true
	}
	for _, todo := range task.Todos {
		if todo.Assignee.AgentID == agentID {
			return true
		}
	}
	return false
}

func copyTodo(todo model.Todo) model.Todo {
	out := todo
	out.Result.Metadata = copyMap(todo.Result.Metadata)
	out.Result.ArtifactRefs = append([]model.TodoResultArtifactRef(nil), todo.Result.ArtifactRefs...)
	return out
}
