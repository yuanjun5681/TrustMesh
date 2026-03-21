package store

import (
	"fmt"
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

type TaskCommentInput struct {
	TaskID  string
	TodoID  string
	Content string
}

func (s *Store) RecordTodoDispatch(userID, taskID, todoID string) (*model.TaskDetail, *transport.AppError) {
	taskID = strings.TrimSpace(taskID)
	todoID = strings.TrimSpace(todoID)
	if taskID == "" || todoID == "" {
		return nil, transport.Validation("invalid todo dispatch payload", map[string]any{"task_id": "required", "todo_id": "required"})
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	task, ok := s.tasks[taskID]
	if !ok || task.UserID != userID {
		return nil, transport.NotFound("task not found")
	}

	todoIdx := findTodoIndex(task, todoID)
	if todoIdx < 0 {
		return nil, transport.NotFound("todo not found")
	}

	todo := &task.Todos[todoIdx]
	if todo.Status != "pending" {
		return nil, transport.Conflict("TODO_NOT_PENDING", "todo is not pending")
	}

	now := time.Now().UTC()
	userName := ""
	if u, ok := s.users[userID]; ok {
		userName = u.Name
	}
	message := fmt.Sprintf("手动派发给 %s", todo.Assignee.Name)
	s.addEventUnsafe(userID, task.ProjectID, task.ID, todo.ID, "user", userID, userName, "todo_assigned", &message, map[string]any{
		"todo_id":           todo.ID,
		"assignee_agent_id": todo.Assignee.AgentID,
		"manual":            true,
	}, now)
	task.UpdatedAt = now
	task.Version++
	if err := s.persistTaskBundleUnsafe(task.ID); err != nil {
		return nil, mongoWriteError(err)
	}
	s.publishTaskUnsafe(task.ID)
	return copyTask(task), nil
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
	s.markAgentSeenUnsafe(pmAgent.ID, now)
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
	s.addEventUnsafe(project.UserID, project.ID, task.ID, "", "agent", pmAgent.ID, pmAgent.Name, "task_created", &taskTitle, map[string]any{"conversation_id": task.ConversationID}, now)
	for _, todo := range task.Todos {
		title := fmt.Sprintf("assigned todo: %s", todo.Title)
		s.addEventUnsafe(project.UserID, project.ID, task.ID, todo.ID, "agent", pmAgent.ID, pmAgent.Name, "todo_assigned", &title, map[string]any{"todo_id": todo.ID, "assignee_agent_id": todo.Assignee.AgentID}, now)
	}

	s.rememberProcessedMessageUnsafe(processedMessageKey("task.create", nodeID, messageID), "task.create", task.ID)
	if err := s.persistConversationUnsafe(conv); err != nil {
		return nil, mongoWriteError(err)
	}
	if err := s.persistTaskBundleUnsafe(task.ID); err != nil {
		return nil, mongoWriteError(err)
	}
	if err := s.persistAgentGraphUnsafe(pmAgent.ID); err != nil {
		return nil, mongoWriteError(err)
	}
	if err := s.persistProcessedMessageUnsafe(processedMessageKey("task.create", nodeID, messageID)); err != nil {
		return nil, mongoWriteError(err)
	}
	s.publishConversationUnsafe(conv.ID)
	s.publishTaskUnsafe(task.ID)

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
	s.markAgentSeenUnsafe(agent.ID, now)
	if todo.Status == "pending" {
		todo.Status = "in_progress"
		todo.StartedAt = &now
		started := fmt.Sprintf("todo started: %s", todo.Title)
		s.addEventUnsafe(task.UserID, task.ProjectID, task.ID, todo.ID, "agent", agent.ID, agent.Name, "todo_started", &started, map[string]any{"todo_id": todo.ID}, now)
	}
	progress := in.Message
	s.addEventUnsafe(task.UserID, task.ProjectID, task.ID, todo.ID, "agent", agent.ID, agent.Name, "todo_progress", &progress, map[string]any{"todo_id": todo.ID}, now)

	s.updateTaskStatusUnsafe(task, now)
	if err := s.persistTaskBundleUnsafe(task.ID); err != nil {
		return nil, mongoWriteError(err)
	}
	s.refreshAgentExecutionStatusUnsafe(agent.ID, now)
	if err := s.persistAgentGraphUnsafe(agent.ID); err != nil {
		return nil, mongoWriteError(err)
	}
	s.publishTaskUnsafe(task.ID)
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
	s.markAgentSeenUnsafe(agent.ID, now)
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
	s.addEventUnsafe(task.UserID, task.ProjectID, task.ID, todo.ID, "agent", agent.ID, agent.Name, "todo_completed", &completed, map[string]any{"todo_id": todo.ID}, now)

	s.updateTaskStatusUnsafe(task, now)
	s.rememberProcessedMessageUnsafe(processedMessageKey("todo.complete", nodeID, messageID), "todo.complete", task.ID)
	if err := s.persistTaskBundleUnsafe(task.ID); err != nil {
		return nil, mongoWriteError(err)
	}
	s.refreshAgentExecutionStatusUnsafe(agent.ID, now)
	if err := s.persistAgentGraphUnsafe(agent.ID); err != nil {
		return nil, mongoWriteError(err)
	}
	if err := s.persistProcessedMessageUnsafe(processedMessageKey("todo.complete", nodeID, messageID)); err != nil {
		return nil, mongoWriteError(err)
	}
	s.publishTaskUnsafe(task.ID)
	return copyTask(task), nil
}

func (s *Store) FailTodoByNode(nodeID string, in TodoFailInput) (*model.TaskDetail, *transport.AppError) {
	return s.FailTodoByNodeWithMessageID(nodeID, "", in)
}

func (s *Store) FailTodoByNodeWithMessageID(nodeID, messageID string, in TodoFailInput) (*model.TaskDetail, *transport.AppError) {
	in.TaskID = strings.TrimSpace(in.TaskID)
	in.TodoID = strings.TrimSpace(in.TodoID)
	in.Error = strings.TrimSpace(in.Error)
	if in.TaskID == "" || in.TodoID == "" || in.Error == "" {
		return nil, transport.Validation("invalid todo.fail payload", map[string]any{"task_id": "required", "todo_id": "required", "error": "required"})
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if task, ok := s.findProcessedTaskUnsafe(processedMessageKey("todo.fail", nodeID, messageID)); ok {
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
	s.markAgentSeenUnsafe(agent.ID, now)
	if todo.StartedAt == nil {
		todo.StartedAt = &now
	}
	todo.Status = "failed"
	todo.CompletedAt = nil
	todo.FailedAt = &now
	errCopy := in.Error
	todo.Error = &errCopy
	failed := fmt.Sprintf("todo failed: %s", todo.Title)
	s.addEventUnsafe(task.UserID, task.ProjectID, task.ID, todo.ID, "agent", agent.ID, agent.Name, "todo_failed", &failed, map[string]any{"todo_id": todo.ID, "error": in.Error}, now)

	s.updateTaskStatusUnsafe(task, now)
	s.rememberProcessedMessageUnsafe(processedMessageKey("todo.fail", nodeID, messageID), "todo.fail", task.ID)
	if err := s.persistTaskBundleUnsafe(task.ID); err != nil {
		return nil, mongoWriteError(err)
	}
	s.refreshAgentExecutionStatusUnsafe(agent.ID, now)
	if err := s.persistAgentGraphUnsafe(agent.ID); err != nil {
		return nil, mongoWriteError(err)
	}
	if err := s.persistProcessedMessageUnsafe(processedMessageKey("todo.fail", nodeID, messageID)); err != nil {
		return nil, mongoWriteError(err)
	}
	s.publishTaskUnsafe(task.ID)
	return copyTask(task), nil
}

func (s *Store) AddTaskCommentByNode(nodeID string, in TaskCommentInput) (*model.Comment, *transport.AppError) {
	in.TaskID = strings.TrimSpace(in.TaskID)
	in.TodoID = strings.TrimSpace(in.TodoID)
	in.Content = strings.TrimSpace(in.Content)
	if in.TaskID == "" || in.Content == "" {
		return nil, transport.Validation("invalid task.comment payload", map[string]any{"task_id": "required", "content": "required"})
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
	if in.TodoID != "" {
		if findTodoIndex(task, in.TodoID) < 0 {
			return nil, transport.NotFound("todo not found")
		}
	}

	now := time.Now().UTC()
	s.markAgentSeenUnsafe(agent.ID, now)
	comment := s.addCommentUnsafe(task, in.TodoID, "agent", agent.ID, agent.Name, in.Content, now)
	if err := s.persistCommentUnsafe(comment); err != nil {
		return nil, mongoWriteError(err)
	}
	s.publishTaskUnsafe(task.ID)
	return comment, nil
}

func (s *Store) AddTaskComment(userID, taskID string, in TaskCommentInput) (*model.Comment, *transport.AppError) {
	taskID = strings.TrimSpace(taskID)
	in.Content = strings.TrimSpace(in.Content)
	in.TodoID = strings.TrimSpace(in.TodoID)
	if taskID == "" || in.Content == "" {
		return nil, transport.Validation("invalid comment payload", map[string]any{"task_id": "required", "content": "required"})
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	task, ok := s.tasks[taskID]
	if !ok || task.UserID != userID {
		return nil, transport.NotFound("task not found")
	}
	if in.TodoID != "" {
		if findTodoIndex(task, in.TodoID) < 0 {
			return nil, transport.NotFound("todo not found")
		}
	}

	now := time.Now().UTC()
	userName := ""
	if u, ok := s.users[userID]; ok {
		userName = u.Name
	}
	comment := s.addCommentUnsafe(task, in.TodoID, "user", userID, userName, in.Content, now)
	if err := s.persistCommentUnsafe(comment); err != nil {
		return nil, mongoWriteError(err)
	}
	s.publishTaskUnsafe(task.ID)
	return comment, nil
}

func (s *Store) ListTaskComments(userID, taskID string) ([]model.Comment, *transport.AppError) {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return nil, transport.Validation("task_id required", nil)
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	task, ok := s.tasks[taskID]
	if !ok || task.UserID != userID {
		return nil, transport.NotFound("task not found")
	}
	comments := s.taskComments[taskID]
	out := make([]model.Comment, len(comments))
	copy(out, comments)
	return out, nil
}

func (s *Store) addCommentUnsafe(task *model.TaskDetail, todoID, actorType, actorID, actorName, content string, at time.Time) *model.Comment {
	comment := &model.Comment{
		ID:        newID(),
		UserID:    task.UserID,
		TaskID:    task.ID,
		TodoID:    todoID,
		ActorType: actorType,
		ActorID:   actorID,
		ActorName: actorName,
		Content:   content,
		CreatedAt: at,
	}
	s.taskComments[task.ID] = append(s.taskComments[task.ID], *comment)
	s.publishUserEventUnsafe(task.UserID, "task.comment.created", map[string]any{
		"task_id":    task.ID,
		"project_id": task.ProjectID,
		"comment":    *comment,
	}, at)

	// 只有 Agent 评论才添加到活动时间线，用户评论不进入 events
	if actorType == "agent" {
		brief := content
		if len(brief) > 80 {
			brief = brief[:80] + "..."
		}
		metadata := map[string]any{"comment_id": comment.ID}
		if todoID != "" {
			metadata["todo_id"] = todoID
		}
		s.addEventUnsafe(task.UserID, task.ProjectID, task.ID, todoID, actorType, actorID, actorName, "task_comment", &brief, metadata, at)
	}

	task.UpdatedAt = at
	task.Version++
	return comment
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
		s.addEventUnsafe(task.UserID, task.ProjectID, task.ID, "", "system", "system", "System", "task_status_changed", &msg, map[string]any{"from": prev, "to": next}, now)
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
		transfersByID := indexTodoTransfers(todo.Result.Metadata)
		for i, ref := range todo.Result.ArtifactRefs {
			baseID := strings.TrimSpace(ref.ArtifactID)
			if baseID == "" {
				baseID = fmt.Sprintf("%s-artifact-%d", todo.ID, i+1)
			}
			artifactID := uniqueTaskArtifactID(baseID, usedIDs)
			sourceTodoID := todo.ID
			metadata := map[string]any{
				"source": "todo_result_ref",
			}
			uri := ""
			var mimeType *string
			if transfer, ok := transfersByID[baseID]; ok {
				metadata["transfer"] = transfer
				metadata["transfer_id"] = baseID
				metadata["ref_only"] = false
				uri = "transfer://" + baseID
				if v := stringValue(transfer["fileName"]); v != "" {
					metadata["file_name"] = v
				} else if v := stringValue(transfer["file_name"]); v != "" {
					metadata["file_name"] = v
				}
				if v := stringValue(transfer["localPath"]); v != "" {
					metadata["local_path"] = v
				} else if v := stringValue(transfer["local_path"]); v != "" {
					metadata["local_path"] = v
				}
				if v := stringValue(transfer["mime_type"]); v != "" {
					mime := v
					mimeType = &mime
				} else if v := stringValue(transfer["mimeType"]); v != "" {
					mime := v
					mimeType = &mime
				}
			} else {
				metadata["ref_only"] = true
			}
			artifacts = append(artifacts, model.TaskArtifact{
				ID:           artifactID,
				SourceTodoID: &sourceTodoID,
				Kind:         strings.TrimSpace(ref.Kind),
				Title:        strings.TrimSpace(ref.Label),
				URI:          uri,
				MimeType:     mimeType,
				Metadata:     metadata,
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

func indexTodoTransfers(metadata map[string]any) map[string]map[string]any {
	out := make(map[string]map[string]any)
	if len(metadata) == 0 {
		return out
	}
	rawTransfers, ok := metadata["transfers"]
	if !ok {
		return out
	}
	items, ok := rawTransfers.([]any)
	if !ok {
		return out
	}
	for _, item := range items {
		transfer, ok := item.(map[string]any)
		if !ok {
			continue
		}
		transferID := strings.TrimSpace(stringValue(transfer["transfer_id"]))
		if transferID == "" {
			transferID = strings.TrimSpace(stringValue(transfer["transferId"]))
		}
		if transferID == "" {
			continue
		}
		out[transferID] = copyMap(transfer)
	}
	return out
}

func stringValue(v any) string {
	s, _ := v.(string)
	return s
}

func findTodoIndex(task *model.TaskDetail, todoID string) int {
	for i := range task.Todos {
		if task.Todos[i].ID == todoID {
			return i
		}
	}
	return -1
}
