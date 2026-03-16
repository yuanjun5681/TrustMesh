package store

import (
	"testing"
	"time"

	"trustmesh/backend/internal/model"
)

func TestGetTaskByConversationForPMNode(t *testing.T) {
	s, _, pm, developer, project, conversation := seedWorkflowState(t)

	task, appErr := s.GetTaskByConversationForPMNode(pm.NodeID, conversation.ID)
	if appErr != nil {
		t.Fatalf("unexpected error before task create: %v", appErr)
	}
	if task != nil {
		t.Fatal("expected nil task before task.create")
	}

	createdTask, appErr := s.CreateTaskByPMNode(pm.NodeID, TaskCreateInput{
		ProjectID:      project.ID,
		ConversationID: conversation.ID,
		Title:          "Implement login",
		Description:    "Support email password login",
		Todos: []TaskCreateTodoInput{
			{
				Title:          "Build backend API",
				Description:    "Implement auth endpoints",
				AssigneeNodeID: developer.NodeID,
			},
		},
	})
	if appErr != nil {
		t.Fatalf("create task: %v", appErr)
	}

	loadedTask, appErr := s.GetTaskByConversationForPMNode(pm.NodeID, conversation.ID)
	if appErr != nil {
		t.Fatalf("unexpected error after task create: %v", appErr)
	}
	if loadedTask == nil {
		t.Fatal("expected task after task.create")
	}
	if loadedTask.ID != createdTask.ID {
		t.Fatalf("unexpected task id: got %s want %s", loadedTask.ID, createdTask.ID)
	}

	otherUser, err := s.CreateUser("other@example.com", "Other", "hash")
	if err != nil {
		t.Fatalf("create other user: %v", err)
	}
	otherPM, err := s.CreateAgent(otherUser.ID, "node-pm-other", "Other PM", "pm", "other", []string{"plan"})
	if err != nil {
		t.Fatalf("create other pm: %v", err)
	}

	_, appErr = s.GetTaskByConversationForPMNode(otherPM.NodeID, conversation.ID)
	if appErr == nil || appErr.Code != "FORBIDDEN" {
		t.Fatalf("expected FORBIDDEN for unrelated pm, got %#v", appErr)
	}

	_, appErr = s.GetTaskByConversationForPMNode(developer.NodeID, conversation.ID)
	if appErr == nil || appErr.Code != "FORBIDDEN" {
		t.Fatalf("expected FORBIDDEN for non-pm, got %#v", appErr)
	}
}

func TestGetProjectSummaryForPMNode(t *testing.T) {
	s, _, pm, developer, project, conversation := seedWorkflowState(t)

	_, appErr := s.CreateTaskByPMNode(pm.NodeID, TaskCreateInput{
		ProjectID:      project.ID,
		ConversationID: conversation.ID,
		Title:          "Implement login",
		Description:    "Support email password login",
		Todos: []TaskCreateTodoInput{
			{
				Title:          "Build backend API",
				Description:    "Implement auth endpoints",
				AssigneeNodeID: developer.NodeID,
			},
		},
	})
	if appErr != nil {
		t.Fatalf("create task: %v", appErr)
	}

	summary, appErr := s.GetProjectSummaryForPMNode(pm.NodeID, project.ID)
	if appErr != nil {
		t.Fatalf("get project summary: %v", appErr)
	}
	if summary.TaskCount != 1 {
		t.Fatalf("unexpected task count: %d", summary.TaskCount)
	}
	if summary.PendingTaskCount != 1 {
		t.Fatalf("unexpected pending task count: %d", summary.PendingTaskCount)
	}
	if summary.ResolvedConversationCount != 1 {
		t.Fatalf("unexpected resolved conversation count: %d", summary.ResolvedConversationCount)
	}
	if summary.ActiveConversationCount != 0 {
		t.Fatalf("unexpected active conversation count: %d", summary.ActiveConversationCount)
	}
}

func TestListCandidateAgentsForPMNode(t *testing.T) {
	s, user, pm, developer, project, _ := seedWorkflowState(t)

	reviewer, appErr := s.CreateAgent(user, "node-review-001", "Reviewer", "reviewer", "review", []string{"review"})
	if appErr != nil {
		t.Fatalf("create reviewer: %v", appErr)
	}
	otherUser, appErr := s.CreateUser("other@example.com", "Other", "hash")
	if appErr != nil {
		t.Fatalf("create other user: %v", appErr)
	}
	_, appErr = s.CreateAgent(otherUser.ID, "node-ext-001", "External", "developer", "external", []string{"backend"})
	if appErr != nil {
		t.Fatalf("create external agent: %v", appErr)
	}

	items, appErr := s.ListCandidateAgentsForPMNode(pm.NodeID, project.ID)
	if appErr != nil {
		t.Fatalf("list candidate agents: %v", appErr)
	}
	if len(items) != 3 {
		t.Fatalf("unexpected candidate count: %d", len(items))
	}

	foundDeveloper := false
	foundReviewer := false
	for _, item := range items {
		if item.ID == developer.ID {
			foundDeveloper = true
		}
		if item.ID == reviewer.ID {
			foundReviewer = true
		}
	}
	if !foundDeveloper || !foundReviewer {
		t.Fatalf("expected developer and reviewer in candidate list: %+v", items)
	}

	_, appErr = s.ListCandidateAgentsForPMNode(developer.NodeID, project.ID)
	if appErr == nil || appErr.Code != "FORBIDDEN" {
		t.Fatalf("expected FORBIDDEN for non-pm, got %#v", appErr)
	}
}

func TestReconcileAgentStatusesMarksOffline(t *testing.T) {
	s, _, pm, developer, project, conversation := seedWorkflowState(t)
	s.heartbeatTTL = 30 * time.Second

	_, appErr := s.CreateTaskByPMNode(pm.NodeID, TaskCreateInput{
		ProjectID:      project.ID,
		ConversationID: conversation.ID,
		Title:          "Implement login",
		Description:    "Support email password login",
		Todos: []TaskCreateTodoInput{
			{
				Title:          "Build backend API",
				Description:    "Implement auth endpoints",
				AssigneeNodeID: developer.NodeID,
			},
		},
	})
	if appErr != nil {
		t.Fatalf("create task: %v", appErr)
	}

	staleAt := time.Now().UTC().Add(-31 * time.Second)
	s.mu.Lock()
	pmAgent := s.agents[pm.ID]
	pmAgent.HeartbeatAt = &staleAt
	pmAgent.Status = "busy"
	developerAgent := s.agents[developer.ID]
	developerAgent.HeartbeatAt = &staleAt
	developerAgent.Status = "online"
	s.mu.Unlock()

	updated := s.ReconcileAgentStatuses(time.Now().UTC())
	if updated != 2 {
		t.Fatalf("unexpected updated count: %d", updated)
	}

	projectState, appErr := s.GetProject(pmAgent.UserID, project.ID)
	if appErr != nil {
		t.Fatalf("get project: %v", appErr)
	}
	if projectState.PMAgent.Status != "offline" {
		t.Fatalf("expected project pm to be offline, got %s", projectState.PMAgent.Status)
	}

	taskState, appErr := s.GetTask(pmAgent.UserID, s.projectTasks[project.ID][0])
	if appErr != nil {
		t.Fatalf("get task: %v", appErr)
	}
	if taskState.PMAgent.Status != "offline" {
		t.Fatalf("expected task pm to be offline, got %s", taskState.PMAgent.Status)
	}
	if taskState.Todos[0].Assignee.NodeID != developer.NodeID {
		t.Fatalf("unexpected todo assignee node id: %s", taskState.Todos[0].Assignee.NodeID)
	}
}

func TestTaskCreateIdempotencyByMessageID(t *testing.T) {
	s, _, pm, developer, project, conversation := seedWorkflowState(t)

	task1, appErr := s.CreateTaskByPMNodeWithMessageID(pm.NodeID, "msg-task-create-1", TaskCreateInput{
		ProjectID:      project.ID,
		ConversationID: conversation.ID,
		Title:          "Implement login",
		Description:    "Support email password login",
		Todos: []TaskCreateTodoInput{
			{
				Title:          "Build backend API",
				Description:    "Implement auth endpoints",
				AssigneeNodeID: developer.NodeID,
			},
		},
	})
	if appErr != nil {
		t.Fatalf("first task.create: %v", appErr)
	}

	task2, appErr := s.CreateTaskByPMNodeWithMessageID(pm.NodeID, "msg-task-create-1", TaskCreateInput{
		ProjectID:      project.ID,
		ConversationID: conversation.ID,
		Title:          "Implement login duplicate",
		Description:    "duplicate retry",
		Todos: []TaskCreateTodoInput{
			{
				Title:          "Build backend API duplicate",
				Description:    "retry",
				AssigneeNodeID: developer.NodeID,
			},
		},
	})
	if appErr != nil {
		t.Fatalf("duplicate task.create should be ignored, got: %v", appErr)
	}
	if task1.ID != task2.ID {
		t.Fatalf("expected same task id for duplicate message: %s vs %s", task1.ID, task2.ID)
	}
	if len(s.projectTasks[project.ID]) != 1 {
		t.Fatalf("expected single task record, got %d", len(s.projectTasks[project.ID]))
	}
}

func TestTodoCompleteIdempotencyByMessageID(t *testing.T) {
	s, _, pm, developer, project, conversation := seedWorkflowState(t)

	task, appErr := s.CreateTaskByPMNode(pm.NodeID, TaskCreateInput{
		ProjectID:      project.ID,
		ConversationID: conversation.ID,
		Title:          "Implement login",
		Description:    "Support email password login",
		Todos: []TaskCreateTodoInput{
			{
				ID:             "todo-1",
				Title:          "Build backend API",
				Description:    "Implement auth endpoints",
				AssigneeNodeID: developer.NodeID,
			},
		},
	})
	if appErr != nil {
		t.Fatalf("create task: %v", appErr)
	}

	result := model.TodoResult{
		Summary: "done",
		Output:  "implemented",
		ArtifactRefs: []model.TodoResultArtifactRef{
			{
				ArtifactID: "artifact-login-api",
				Kind:       "report",
				Label:      "Login API report",
			},
		},
		Metadata: map[string]any{"duration_ms": 100},
	}
	task1, appErr := s.CompleteTodoByNodeWithMessageID(developer.NodeID, "msg-todo-complete-1", TodoCompleteInput{
		TaskID: task.ID,
		TodoID: "todo-1",
		Result: result,
	})
	if appErr != nil {
		t.Fatalf("first todo.complete: %v", appErr)
	}
	task2, appErr := s.CompleteTodoByNodeWithMessageID(developer.NodeID, "msg-todo-complete-1", TodoCompleteInput{
		TaskID: task.ID,
		TodoID: "todo-1",
		Result: result,
	})
	if appErr != nil {
		t.Fatalf("duplicate todo.complete should be ignored, got: %v", appErr)
	}
	if task2.Status != "done" || task1.ID != task2.ID {
		t.Fatalf("unexpected duplicate todo.complete result: task1=%s task2=%s status=%s", task1.ID, task2.ID, task2.Status)
	}
	if len(task2.Artifacts) != 1 {
		t.Fatalf("expected a single aggregated artifact, got %d", len(task2.Artifacts))
	}
	if task2.Result.Metadata["artifact_count"] != 1 {
		t.Fatalf("expected artifact_count=1, got %#v", task2.Result.Metadata["artifact_count"])
	}

	events, appErr := s.ListTaskEvents(task1.UserID, task.ID)
	if appErr != nil {
		t.Fatalf("list task events: %v", appErr)
	}
	completedEvents := 0
	for _, event := range events {
		if event.EventType == "todo_completed" {
			completedEvents++
		}
	}
	if completedEvents != 1 {
		t.Fatalf("expected a single todo_completed event, got %d", completedEvents)
	}
}

func TestTaskResultAggregationFromCompletedTodos(t *testing.T) {
	s, _, pm, developer, project, conversation := seedWorkflowState(t)

	task, appErr := s.CreateTaskByPMNode(pm.NodeID, TaskCreateInput{
		ProjectID:      project.ID,
		ConversationID: conversation.ID,
		Title:          "Implement login",
		Description:    "Support email password login",
		Todos: []TaskCreateTodoInput{
			{
				ID:             "todo-1",
				Title:          "Build backend API",
				Description:    "Implement auth endpoints",
				AssigneeNodeID: developer.NodeID,
			},
			{
				ID:             "todo-2",
				Title:          "Write rollout notes",
				Description:    "Document changes",
				AssigneeNodeID: developer.NodeID,
			},
		},
	})
	if appErr != nil {
		t.Fatalf("create task: %v", appErr)
	}

	task, appErr = s.CompleteTodoByNode(developer.NodeID, TodoCompleteInput{
		TaskID: task.ID,
		TodoID: "todo-1",
		Result: model.TodoResult{
			Summary: "API ready",
			Output:  "Added register/login endpoints",
			ArtifactRefs: []model.TodoResultArtifactRef{
				{
					ArtifactID: "artifact-login-api",
					Kind:       "report",
					Label:      "Login API report",
				},
			},
			Metadata: map[string]any{"duration_ms": 1200},
		},
	})
	if appErr != nil {
		t.Fatalf("complete first todo: %v", appErr)
	}

	if task.Status != "in_progress" {
		t.Fatalf("expected in_progress after first completion, got %s", task.Status)
	}
	if len(task.Artifacts) != 1 {
		t.Fatalf("expected 1 artifact after first completion, got %d", len(task.Artifacts))
	}
	if task.Result.Metadata["completed_todo_count"] != 1 {
		t.Fatalf("expected completed_todo_count=1, got %#v", task.Result.Metadata["completed_todo_count"])
	}
	if task.Result.Metadata["pending_todo_count"] != 1 {
		t.Fatalf("expected pending_todo_count=1, got %#v", task.Result.Metadata["pending_todo_count"])
	}

	task, appErr = s.CompleteTodoByNode(developer.NodeID, TodoCompleteInput{
		TaskID: task.ID,
		TodoID: "todo-2",
		Result: model.TodoResult{
			Summary: "Docs ready",
			Output:  "Added rollout checklist",
			ArtifactRefs: []model.TodoResultArtifactRef{
				{
					ArtifactID: "artifact-rollout-notes",
					Kind:       "report",
					Label:      "Rollout notes",
				},
			},
			Metadata: map[string]any{"duration_ms": 400},
		},
	})
	if appErr != nil {
		t.Fatalf("complete second todo: %v", appErr)
	}

	if task.Status != "done" {
		t.Fatalf("expected done after all todos complete, got %s", task.Status)
	}
	if len(task.Artifacts) != 2 {
		t.Fatalf("expected 2 artifacts, got %d", len(task.Artifacts))
	}
	if task.Artifacts[0].ID != "artifact-login-api" || task.Artifacts[1].ID != "artifact-rollout-notes" {
		t.Fatalf("unexpected artifact ids: %+v", task.Artifacts)
	}
	if task.Result.Summary == "" || task.Result.FinalOutput == "" {
		t.Fatalf("expected aggregated task result, got %+v", task.Result)
	}
	if task.Result.Metadata["artifact_count"] != 2 {
		t.Fatalf("expected artifact_count=2, got %#v", task.Result.Metadata["artifact_count"])
	}
	if task.Result.Metadata["completed_todo_count"] != 2 {
		t.Fatalf("expected completed_todo_count=2, got %#v", task.Result.Metadata["completed_todo_count"])
	}
}

func TestTaskResultAggregationOnTodoFailure(t *testing.T) {
	s, _, pm, developer, project, conversation := seedWorkflowState(t)

	task, appErr := s.CreateTaskByPMNode(pm.NodeID, TaskCreateInput{
		ProjectID:      project.ID,
		ConversationID: conversation.ID,
		Title:          "Implement login",
		Description:    "Support email password login",
		Todos: []TaskCreateTodoInput{
			{
				ID:             "todo-1",
				Title:          "Build backend API",
				Description:    "Implement auth endpoints",
				AssigneeNodeID: developer.NodeID,
			},
		},
	})
	if appErr != nil {
		t.Fatalf("create task: %v", appErr)
	}

	task, appErr = s.FailTodoByNode(developer.NodeID, TodoFailInput{
		TaskID: task.ID,
		TodoID: "todo-1",
		Error:  "missing oauth credentials",
	})
	if appErr != nil {
		t.Fatalf("fail todo: %v", appErr)
	}

	if task.Status != "failed" {
		t.Fatalf("expected failed task, got %s", task.Status)
	}
	if task.Result.Summary == "" || task.Result.FinalOutput == "" {
		t.Fatalf("expected failed task aggregation, got %+v", task.Result)
	}
	if task.Result.Metadata["failed_todo_count"] != 1 {
		t.Fatalf("expected failed_todo_count=1, got %#v", task.Result.Metadata["failed_todo_count"])
	}
	if len(task.Artifacts) != 0 {
		t.Fatalf("expected no artifacts on failed todo, got %d", len(task.Artifacts))
	}
}

func seedWorkflowState(t *testing.T) (*Store, string, stringAgent, stringAgent, projectRef, conversationRef) {
	t.Helper()

	s := New()
	user, appErr := s.CreateUser("user@example.com", "User", "hash")
	if appErr != nil {
		t.Fatalf("create user: %v", appErr)
	}
	pm, appErr := s.CreateAgent(user.ID, "node-pm-001", "PM Agent", "pm", "pm", []string{"plan"})
	if appErr != nil {
		t.Fatalf("create pm: %v", appErr)
	}
	developer, appErr := s.CreateAgent(user.ID, "node-dev-001", "Developer", "developer", "dev", []string{"backend"})
	if appErr != nil {
		t.Fatalf("create developer: %v", appErr)
	}
	project, appErr := s.CreateProject(user.ID, "TrustMesh MVP", "demo", pm.ID)
	if appErr != nil {
		t.Fatalf("create project: %v", appErr)
	}
	conversation, appErr := s.CreateConversation(user.ID, project.ID, "Need login")
	if appErr != nil {
		t.Fatalf("create conversation: %v", appErr)
	}

	return s, user.ID, stringAgent{ID: pm.ID, NodeID: pm.NodeID}, stringAgent{ID: developer.ID, NodeID: developer.NodeID}, projectRef{ID: project.ID}, conversationRef{ID: conversation.ID}
}

type stringAgent struct {
	ID     string
	NodeID string
}

type projectRef struct {
	ID string
}

type conversationRef struct {
	ID string
}
