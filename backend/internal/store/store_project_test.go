package store

import "testing"

func TestProjectTaskSummaryReflectsTaskState(t *testing.T) {
	s, userID, pm, developer, project, conversation := seedWorkflowState(t)

	projectState, appErr := s.GetProject(userID, project.ID)
	if appErr != nil {
		t.Fatalf("get empty project: %v", appErr)
	}
	if projectState.TaskSummary.WorkStatus != "empty" {
		t.Fatalf("expected empty work status, got %s", projectState.TaskSummary.WorkStatus)
	}
	if projectState.TaskSummary.TaskTotal != 0 {
		t.Fatalf("expected no tasks, got %d", projectState.TaskSummary.TaskTotal)
	}

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

	projectState, appErr = s.GetProject(userID, project.ID)
	if appErr != nil {
		t.Fatalf("get queued project: %v", appErr)
	}
	if projectState.TaskSummary.WorkStatus != "queued" {
		t.Fatalf("expected queued work status, got %s", projectState.TaskSummary.WorkStatus)
	}
	if projectState.TaskSummary.PendingCount != 1 || projectState.TaskSummary.TaskTotal != 1 {
		t.Fatalf("unexpected queued summary: %#v", projectState.TaskSummary)
	}
	if projectState.TaskSummary.LatestTaskAt == nil {
		t.Fatal("expected latest task timestamp to be set")
	}

	_, appErr = s.RecordSequentialTodoDispatch(task.ID, "todo-1")
	if appErr != nil {
		t.Fatalf("dispatch todo: %v", appErr)
	}

	projectState, appErr = s.GetProject(userID, project.ID)
	if appErr != nil {
		t.Fatalf("get running project: %v", appErr)
	}
	if projectState.TaskSummary.WorkStatus != "running" {
		t.Fatalf("expected running work status, got %s", projectState.TaskSummary.WorkStatus)
	}
	if projectState.TaskSummary.InProgressCount != 1 {
		t.Fatalf("unexpected running summary: %#v", projectState.TaskSummary)
	}

	_, appErr = s.CompleteTodoByNode(developer.NodeID, TodoCompleteInput{
		TaskID: task.ID,
		TodoID: "todo-1",
	})
	if appErr != nil {
		t.Fatalf("complete todo: %v", appErr)
	}

	projectState, appErr = s.GetProject(userID, project.ID)
	if appErr != nil {
		t.Fatalf("get idle project: %v", appErr)
	}
	if projectState.TaskSummary.WorkStatus != "idle" {
		t.Fatalf("expected idle work status, got %s", projectState.TaskSummary.WorkStatus)
	}
	if projectState.TaskSummary.DoneCount != 1 {
		t.Fatalf("unexpected idle summary: %#v", projectState.TaskSummary)
	}
}

func TestProjectTaskSummaryPrioritizesAttentionAndArchive(t *testing.T) {
	s, userID, pm, developer, project, conversation := seedWorkflowState(t)

	task, appErr := s.CreateTaskByPMNode(pm.NodeID, TaskCreateInput{
		ProjectID:      project.ID,
		ConversationID: conversation.ID,
		Title:          "Investigate bug",
		Description:    "Reproduce flaky test",
		Todos: []TaskCreateTodoInput{
			{
				ID:             "todo-1",
				Title:          "Inspect logs",
				Description:    "Collect the latest failure logs",
				AssigneeNodeID: developer.NodeID,
			},
		},
	})
	if appErr != nil {
		t.Fatalf("create task: %v", appErr)
	}

	_, appErr = s.FailTodoByNode(developer.NodeID, TodoFailInput{
		TaskID: task.ID,
		TodoID: "todo-1",
		Error:  "tests are still flaky",
	})
	if appErr != nil {
		t.Fatalf("fail todo: %v", appErr)
	}

	projectState, appErr := s.GetProject(userID, project.ID)
	if appErr != nil {
		t.Fatalf("get attention project: %v", appErr)
	}
	if projectState.TaskSummary.WorkStatus != "attention" {
		t.Fatalf("expected attention work status, got %s", projectState.TaskSummary.WorkStatus)
	}
	if projectState.TaskSummary.FailedCount != 1 {
		t.Fatalf("unexpected attention summary: %#v", projectState.TaskSummary)
	}

	projectState, appErr = s.ArchiveProject(userID, project.ID)
	if appErr != nil {
		t.Fatalf("archive project: %v", appErr)
	}
	if projectState.TaskSummary.WorkStatus != "archived" {
		t.Fatalf("expected archived work status, got %s", projectState.TaskSummary.WorkStatus)
	}
}
