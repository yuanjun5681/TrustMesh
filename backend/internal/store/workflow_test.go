package store

import (
	"testing"
	"time"

	"trustmesh/backend/internal/model"
)

func TestSyncAgentPresenceMarksOfflineAndBusy(t *testing.T) {
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

	updated := s.SyncAgentPresence([]AgentPresence{
		{NodeID: developer.NodeID, LastSeenAt: time.Now().UTC()},
	}, time.Now().UTC())
	if updated != 2 {
		t.Fatalf("unexpected updated count: %d", updated)
	}

	projectState, appErr := s.GetProject(s.agents[pm.ID].UserID, project.ID)
	if appErr != nil {
		t.Fatalf("get project: %v", appErr)
	}
	if projectState.PMAgent.Status != "offline" {
		t.Fatalf("expected project pm to be offline, got %s", projectState.PMAgent.Status)
	}

	taskState, appErr := s.GetTask(s.agents[pm.ID].UserID, s.projectTasks[project.ID][0])
	if appErr != nil {
		t.Fatalf("get task: %v", appErr)
	}
	if taskState.PMAgent.Status != "offline" {
		t.Fatalf("expected task pm to be offline, got %s", taskState.PMAgent.Status)
	}
	if s.agents[developer.ID].Status != "online" {
		t.Fatalf("expected developer to stay online before starting work, got %s", s.agents[developer.ID].Status)
	}

	taskState, appErr = s.UpdateTodoProgressByNode(developer.NodeID, TodoProgressInput{
		TaskID:  taskState.ID,
		TodoID:  taskState.Todos[0].ID,
		Message: "started",
	})
	if appErr != nil {
		t.Fatalf("todo progress: %v", appErr)
	}
	if s.agents[developer.ID].Status != "busy" {
		t.Fatalf("expected developer to be busy after progress, got %s", s.agents[developer.ID].Status)
	}
	if taskState.Todos[0].Assignee.NodeID != developer.NodeID {
		t.Fatalf("unexpected todo assignee node id: %s", taskState.Todos[0].Assignee.NodeID)
	}
}

func TestAgentUsageAndDeleteConflictDetails(t *testing.T) {
	s, userID, pm, developer, project, conversation := seedWorkflowState(t)

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

	agents := s.ListAgents(userID)
	if len(agents) != 2 {
		t.Fatalf("expected 2 agents, got %d", len(agents))
	}

	var pmUsage, developerUsage model.AgentUsage
	for _, agent := range agents {
		switch agent.ID {
		case pm.ID:
			pmUsage = agent.Usage
		case developer.ID:
			developerUsage = agent.Usage
		}
	}

	if !pmUsage.InUse || pmUsage.ProjectCount != 1 || pmUsage.TaskCount != 1 || pmUsage.TodoCount != 0 || pmUsage.TotalCount != 2 {
		t.Fatalf("unexpected pm usage: %#v", pmUsage)
	}
	if !developerUsage.InUse || developerUsage.ProjectCount != 0 || developerUsage.TaskCount != 0 || developerUsage.TodoCount != 1 || developerUsage.TotalCount != 1 {
		t.Fatalf("unexpected developer usage: %#v", developerUsage)
	}

	agent, appErr := s.GetAgent(userID, developer.ID)
	if appErr != nil {
		t.Fatalf("get agent: %v", appErr)
	}
	if !agent.Usage.InUse || agent.Usage.TodoCount != 1 {
		t.Fatalf("unexpected get agent usage: %#v", agent.Usage)
	}

	// 删除有引用的 agent 应该软删除（归档）而非报错
	appErr = s.DeleteAgent(userID, developer.ID)
	if appErr != nil {
		t.Fatalf("expected soft delete to succeed, got: %v", appErr)
	}

	// 归档后 ListAgents 不应包含该 agent
	agentsAfterArchive := s.ListAgents(userID)
	for _, a := range agentsAfterArchive {
		if a.ID == developer.ID {
			t.Fatal("archived agent should not appear in ListAgents")
		}
	}

	// GetAgent 仍可查看归档 agent
	archivedAgent, appErr := s.GetAgent(userID, developer.ID)
	if appErr != nil {
		t.Fatalf("GetAgent on archived agent should succeed: %v", appErr)
	}
	if !archivedAgent.Archived {
		t.Fatal("expected agent to be archived")
	}

	// 再次删除归档 agent 应返回 not found
	appErr = s.DeleteAgent(userID, developer.ID)
	if appErr == nil || appErr.Status != 404 {
		t.Fatalf("expected not found for already archived agent, got: %v", appErr)
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

func TestTaskArtifactAggregationIncludesTransferMetadata(t *testing.T) {
	s, _, pm, developer, project, conversation := seedWorkflowState(t)

	task, appErr := s.CreateTaskByPMNode(pm.NodeID, TaskCreateInput{
		ProjectID:      project.ID,
		ConversationID: conversation.ID,
		Title:          "Deliver report",
		Description:    "Upload final report",
		Todos: []TaskCreateTodoInput{
			{
				ID:             "todo-1",
				Title:          "Upload report",
				Description:    "Send report PDF",
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
			Summary: "Report uploaded",
			Output:  "Uploaded the final PDF report",
			ArtifactRefs: []model.TodoResultArtifactRef{
				{
					ArtifactID: "tf_report_123",
					Kind:       "file",
					Label:      "Final report PDF",
				},
			},
			Metadata: map[string]any{
				"transfers": []any{
					map[string]any{
						"transfer_id": "tf_report_123",
						"bucket":      "deliverables",
						"size":        2048,
						"checksum":    "sha256:abc123",
						"mime_type":   "application/pdf",
						"fileName":    "report.pdf",
						"localPath":   "/tmp/report.pdf",
					},
				},
			},
		},
	})
	if appErr != nil {
		t.Fatalf("complete todo: %v", appErr)
	}

	if len(task.Artifacts) != 1 {
		t.Fatalf("expected 1 artifact, got %d", len(task.Artifacts))
	}
	artifact := task.Artifacts[0]
	if artifact.URI != "transfer://tf_report_123" {
		t.Fatalf("unexpected artifact uri: %s", artifact.URI)
	}
	if artifact.MimeType == nil || *artifact.MimeType != "application/pdf" {
		t.Fatalf("unexpected mime type: %#v", artifact.MimeType)
	}
	if artifact.Metadata["transfer_id"] != "tf_report_123" {
		t.Fatalf("unexpected transfer_id metadata: %#v", artifact.Metadata["transfer_id"])
	}
	if artifact.Metadata["file_name"] != "report.pdf" {
		t.Fatalf("unexpected file_name metadata: %#v", artifact.Metadata["file_name"])
	}
	if artifact.Metadata["local_path"] != "/tmp/report.pdf" {
		t.Fatalf("unexpected local_path metadata: %#v", artifact.Metadata["local_path"])
	}
	transfer, ok := artifact.Metadata["transfer"].(map[string]any)
	if !ok {
		t.Fatalf("expected transfer metadata map, got %#v", artifact.Metadata["transfer"])
	}
	if transfer["bucket"] != "deliverables" {
		t.Fatalf("unexpected transfer metadata: %#v", transfer)
	}
}

func TestRecordTodoDispatchMarksTaskInProgress(t *testing.T) {
	s, userID, pm, developer, project, conversation := seedWorkflowState(t)

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

	task, appErr = s.RecordTodoDispatch(userID, task.ID, "todo-1")
	if appErr != nil {
		t.Fatalf("dispatch todo: %v", appErr)
	}

	if task.Status != "in_progress" {
		t.Fatalf("expected task in_progress after dispatch, got %s", task.Status)
	}
	if task.Todos[0].Status != "in_progress" {
		t.Fatalf("expected todo in_progress after dispatch, got %s", task.Todos[0].Status)
	}
	if task.Todos[0].StartedAt == nil {
		t.Fatal("expected todo started_at to be set after dispatch")
	}
	if s.agents[developer.ID].Status != "busy" {
		t.Fatalf("expected developer to be busy after dispatch, got %s", s.agents[developer.ID].Status)
	}
	if task.Result.Summary == "" {
		t.Fatalf("expected in-progress task result summary after dispatch, got %+v", task.Result)
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

func TestTaskTodoOrderAndSequentialExecutionGuards(t *testing.T) {
	s, _, pm, developer, project, conversation := seedWorkflowState(t)

	task, appErr := s.CreateTaskByPMNode(pm.NodeID, TaskCreateInput{
		ProjectID:      project.ID,
		ConversationID: conversation.ID,
		Title:          "Implement login",
		Description:    "Support email password login",
		Todos: []TaskCreateTodoInput{
			{
				ID:             "todo-2",
				Order:          2,
				Title:          "Build frontend UI",
				Description:    "Implement the login form after backend API is ready",
				AssigneeNodeID: developer.NodeID,
			},
			{
				ID:             "todo-1",
				Order:          1,
				Title:          "Build backend API",
				Description:    "Implement auth endpoints first",
				AssigneeNodeID: developer.NodeID,
			},
		},
	})
	if appErr != nil {
		t.Fatalf("create task: %v", appErr)
	}

	if len(task.Todos) != 2 {
		t.Fatalf("expected 2 todos, got %d", len(task.Todos))
	}
	if task.Todos[0].ID != "todo-1" || task.Todos[0].Order != 1 {
		t.Fatalf("expected first todo to be todo-1/order=1, got %+v", task.Todos[0])
	}
	if task.Todos[1].ID != "todo-2" || task.Todos[1].Order != 2 {
		t.Fatalf("expected second todo to be todo-2/order=2, got %+v", task.Todos[1])
	}

	_, appErr = s.UpdateTodoProgressByNode(developer.NodeID, TodoProgressInput{
		TaskID:  task.ID,
		TodoID:  "todo-2",
		Message: "trying to skip ahead",
	})
	if appErr == nil || appErr.Code != "TODO_BLOCKED_BY_PREVIOUS" {
		t.Fatalf("expected TODO_BLOCKED_BY_PREVIOUS for out-of-order progress, got %#v", appErr)
	}

	task, appErr = s.CompleteTodoByNode(developer.NodeID, TodoCompleteInput{
		TaskID: task.ID,
		TodoID: "todo-1",
		Result: model.TodoResult{
			Summary:  "API ready",
			Output:   "backend done",
			Metadata: map[string]any{},
		},
	})
	if appErr != nil {
		t.Fatalf("complete first todo: %v", appErr)
	}

	task, appErr = s.CompleteTodoByNode(developer.NodeID, TodoCompleteInput{
		TaskID: task.ID,
		TodoID: "todo-2",
		Result: model.TodoResult{
			Summary:  "UI ready",
			Output:   "frontend done",
			Metadata: map[string]any{},
		},
	})
	if appErr != nil {
		t.Fatalf("complete second todo: %v", appErr)
	}

	if task.Status != "done" {
		t.Fatalf("expected task done after ordered completion, got %s", task.Status)
	}
}

func TestCancelTaskStopsFurtherTodoUpdates(t *testing.T) {
	s, userID, pm, developer, project, conversation := seedWorkflowState(t)

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
				Title:          "Write docs",
				Description:    "Document rollout",
				AssigneeNodeID: developer.NodeID,
			},
		},
	})
	if appErr != nil {
		t.Fatalf("create task: %v", appErr)
	}

	task, appErr = s.RecordSequentialTodoDispatch(task.ID, "todo-1")
	if appErr != nil {
		t.Fatalf("dispatch first todo: %v", appErr)
	}
	if task.Status != "in_progress" {
		t.Fatalf("expected in_progress before cancel, got %s", task.Status)
	}

	task, appErr = s.CancelTask(userID, TaskCancelInput{
		TaskID: task.ID,
		Reason: "manual stop",
	})
	if appErr != nil {
		t.Fatalf("cancel task: %v", appErr)
	}
	if task.Status != "canceled" {
		t.Fatalf("expected canceled task status, got %s", task.Status)
	}
	if task.CancelReason == nil || *task.CancelReason != "manual stop" {
		t.Fatalf("unexpected cancel reason: %#v", task.CancelReason)
	}
	if task.CanceledBy == nil || task.CanceledBy.ActorID != userID {
		t.Fatalf("unexpected canceled_by: %#v", task.CanceledBy)
	}
	if task.Todos[0].Status != "canceled" || task.Todos[1].Status != "canceled" {
		t.Fatalf("expected unfinished todos to be canceled: %#v", task.Todos)
	}
	if task.Result.Metadata["canceled_todo_count"] != 2 {
		t.Fatalf("expected canceled_todo_count=2, got %#v", task.Result.Metadata["canceled_todo_count"])
	}

	_, appErr = s.UpdateTodoProgressByNode(developer.NodeID, TodoProgressInput{
		TaskID:  task.ID,
		TodoID:  "todo-1",
		Message: "late progress",
	})
	if appErr == nil || appErr.Code != "TASK_CANCELED" {
		t.Fatalf("expected TASK_CANCELED for progress after cancel, got %#v", appErr)
	}

	_, appErr = s.CompleteTodoByNode(developer.NodeID, TodoCompleteInput{
		TaskID: task.ID,
		TodoID: "todo-1",
		Result: model.TodoResult{Summary: "late", Output: "late"},
	})
	if appErr == nil || appErr.Code != "TASK_CANCELED" {
		t.Fatalf("expected TASK_CANCELED for complete after cancel, got %#v", appErr)
	}

	_, appErr = s.RecordSequentialTodoDispatch(task.ID, "todo-2")
	if appErr == nil || appErr.Code != "TASK_CANCELED" {
		t.Fatalf("expected TASK_CANCELED for dispatch after cancel, got %#v", appErr)
	}
}

func TestCreateTaskByUser(t *testing.T) {
	s, userID, _, developer, project, _ := seedWorkflowState(t)

	task, appErr := s.CreateTaskByUser(userID, UserTaskCreateInput{
		ProjectID:       project.ID,
		Title:           "Implement login",
		Description:     "Support email password login",
		Priority:        "high",
		AssigneeAgentID: developer.ID,
	})
	if appErr != nil {
		t.Fatalf("create task: %v", appErr)
	}
	if task.Title != "Implement login" {
		t.Fatalf("unexpected title: %s", task.Title)
	}
	if task.Status != "pending" {
		t.Fatalf("expected pending, got %s", task.Status)
	}
	if task.Priority != "high" {
		t.Fatalf("expected high priority, got %s", task.Priority)
	}
	if task.ConversationID != "" {
		t.Fatalf("expected empty conversation_id, got %s", task.ConversationID)
	}
	if len(task.Todos) != 1 {
		t.Fatalf("expected 1 todo, got %d", len(task.Todos))
	}
	todo := task.Todos[0]
	if todo.Title != "Implement login" {
		t.Fatalf("todo title should match task title, got %s", todo.Title)
	}
	if todo.Assignee.AgentID != developer.ID {
		t.Fatalf("unexpected assignee: %s", todo.Assignee.AgentID)
	}
	if todo.Status != "pending" {
		t.Fatalf("expected todo pending, got %s", todo.Status)
	}

	// Verify task appears in project tasks
	fetched, appErr := s.GetTask(userID, task.ID)
	if appErr != nil {
		t.Fatalf("get task: %v", appErr)
	}
	if fetched.ID != task.ID {
		t.Fatalf("fetched task id mismatch")
	}
}

func TestCreateTaskByUserValidation(t *testing.T) {
	s, userID, _, developer, project, _ := seedWorkflowState(t)

	// Missing title
	_, appErr := s.CreateTaskByUser(userID, UserTaskCreateInput{
		ProjectID:       project.ID,
		Title:           "",
		Description:     "desc",
		AssigneeAgentID: developer.ID,
	})
	if appErr == nil {
		t.Fatal("expected validation error for missing title")
	}

	// Invalid priority
	_, appErr = s.CreateTaskByUser(userID, UserTaskCreateInput{
		ProjectID:       project.ID,
		Title:           "Test",
		Description:     "desc",
		Priority:        "critical",
		AssigneeAgentID: developer.ID,
	})
	if appErr == nil {
		t.Fatal("expected validation error for invalid priority")
	}

	// Default priority
	task, appErr := s.CreateTaskByUser(userID, UserTaskCreateInput{
		ProjectID:       project.ID,
		Title:           "Test",
		Description:     "desc",
		AssigneeAgentID: developer.ID,
	})
	if appErr != nil {
		t.Fatalf("create task: %v", appErr)
	}
	if task.Priority != "medium" {
		t.Fatalf("expected default medium priority, got %s", task.Priority)
	}

	// Archived project
	s.mu.Lock()
	s.projects[project.ID].Status = "archived"
	s.mu.Unlock()
	_, appErr = s.CreateTaskByUser(userID, UserTaskCreateInput{
		ProjectID:       project.ID,
		Title:           "Test",
		Description:     "desc",
		AssigneeAgentID: developer.ID,
	})
	if appErr == nil || appErr.Code != "PROJECT_ARCHIVED" {
		t.Fatalf("expected PROJECT_ARCHIVED, got %v", appErr)
	}
}

func TestCreateTaskByUserTodoWorkflow(t *testing.T) {
	s, userID, _, developer, project, _ := seedWorkflowState(t)

	task, appErr := s.CreateTaskByUser(userID, UserTaskCreateInput{
		ProjectID:       project.ID,
		Title:           "Build API",
		Description:     "REST endpoints",
		AssigneeAgentID: developer.ID,
	})
	if appErr != nil {
		t.Fatalf("create task: %v", appErr)
	}

	// Dispatch the todo
	task, appErr = s.RecordTodoDispatch(userID, task.ID, task.Todos[0].ID)
	if appErr != nil {
		t.Fatalf("dispatch todo: %v", appErr)
	}
	if task.Status != "in_progress" {
		t.Fatalf("expected in_progress after dispatch, got %s", task.Status)
	}

	// Complete the todo
	task, appErr = s.CompleteTodoByNode(developer.NodeID, TodoCompleteInput{
		TaskID: task.ID,
		TodoID: task.Todos[0].ID,
		Result: model.TodoResult{Summary: "done"},
	})
	if appErr != nil {
		t.Fatalf("complete todo: %v", appErr)
	}
	if task.Status != "done" {
		t.Fatalf("expected done after completing only todo, got %s", task.Status)
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
	s.SyncAgentPresence([]AgentPresence{
		{NodeID: pm.NodeID, LastSeenAt: time.Now().UTC()},
		{NodeID: developer.NodeID, LastSeenAt: time.Now().UTC()},
	}, time.Now().UTC())
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
