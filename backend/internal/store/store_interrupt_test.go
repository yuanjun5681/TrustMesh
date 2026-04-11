package store

import (
	"testing"

	"trustmesh/backend/internal/model"
)

func TestInterruptTaskByAgentMarksInterrupted(t *testing.T) {
	s, userID, pm, developer, project := seedWorkflowState(t)

	task, appErr := s.CreateTaskByPMNode(pm.NodeID, TaskCreateInput{
		ProjectID:   project.ID,
		Title:       "Implement login",
		Description: "Support email password login",
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

	// 让 todo 进入 in_progress
	if _, appErr := s.UpdateTodoProgressByNode(developer.NodeID, TodoProgressInput{
		TaskID:  task.ID,
		TodoID:  "todo-1",
		Message: "started",
	}); appErr != nil {
		t.Fatalf("todo progress: %v", appErr)
	}

	interrupted, appErr := s.InterruptTaskByAgent(developer.NodeID, task.ID, "opencode exited -1")
	if appErr != nil {
		t.Fatalf("interrupt task: %v", appErr)
	}
	if interrupted.Status != "interrupted" {
		t.Fatalf("expected task interrupted, got %s", interrupted.Status)
	}
	if interrupted.InterruptedAt == nil || interrupted.InterruptReason == nil || *interrupted.InterruptReason != "opencode exited -1" {
		t.Fatalf("expected interrupt reason populated, got %#v", interrupted.InterruptReason)
	}
	if interrupted.InterruptCount != 1 {
		t.Fatalf("expected interrupt_count=1, got %d", interrupted.InterruptCount)
	}
	if len(interrupted.Todos) != 1 || interrupted.Todos[0].Status != "interrupted" {
		t.Fatalf("expected todo interrupted, got %#v", interrupted.Todos)
	}
	if interrupted.Todos[0].InterruptReason == nil || *interrupted.Todos[0].InterruptReason != "opencode exited -1" {
		t.Fatalf("expected todo interrupt reason populated, got %#v", interrupted.Todos[0].InterruptReason)
	}

	// 二次中断累加 interrupt_count
	interrupted2, appErr := s.InterruptTaskByAgent(developer.NodeID, task.ID, "agent crashed again")
	if appErr != nil {
		t.Fatalf("second interrupt: %v", appErr)
	}
	if interrupted2.InterruptCount != 2 {
		t.Fatalf("expected interrupt_count=2 after retry, got %d", interrupted2.InterruptCount)
	}

	events, appErr := s.ListTaskEvents(userID, task.ID)
	if appErr != nil {
		t.Fatalf("list events: %v", appErr)
	}
	taskInterruptedCount := 0
	todoInterruptedCount := 0
	for _, event := range events {
		switch event.EventType {
		case "task_interrupted":
			taskInterruptedCount++
		case "todo_interrupted":
			todoInterruptedCount++
		}
	}
	if taskInterruptedCount != 2 {
		t.Fatalf("expected 2 task_interrupted events, got %d", taskInterruptedCount)
	}
	if todoInterruptedCount != 1 {
		// 二次中断时活跃 todo 已经是 interrupted 状态，不会再写一次 todo_interrupted
		t.Fatalf("expected 1 todo_interrupted event, got %d", todoInterruptedCount)
	}
}

func TestResumeTaskByUserResetsInterruptedTodos(t *testing.T) {
	s, userID, pm, developer, project := seedWorkflowState(t)

	task, appErr := s.CreateTaskByPMNode(pm.NodeID, TaskCreateInput{
		ProjectID:   project.ID,
		Title:       "Resumable task",
		Description: "demo",
		Todos: []TaskCreateTodoInput{
			{
				ID:             "todo-1",
				Title:          "Step 1",
				Description:    "first step",
				AssigneeNodeID: developer.NodeID,
			},
		},
	})
	if appErr != nil {
		t.Fatalf("create task: %v", appErr)
	}
	if _, appErr := s.UpdateTodoProgressByNode(developer.NodeID, TodoProgressInput{
		TaskID:  task.ID,
		TodoID:  "todo-1",
		Message: "started",
	}); appErr != nil {
		t.Fatalf("progress: %v", appErr)
	}
	if _, appErr := s.InterruptTaskByAgent(developer.NodeID, task.ID, "agent down"); appErr != nil {
		t.Fatalf("interrupt: %v", appErr)
	}

	resumed, appErr := s.ResumeTaskByUser(userID, task.ID)
	if appErr != nil {
		t.Fatalf("resume task: %v", appErr)
	}
	if resumed.Status != "pending" {
		t.Fatalf("expected resumed task pending, got %s", resumed.Status)
	}
	if resumed.InterruptedAt != nil || resumed.InterruptReason != nil {
		t.Fatalf("expected interrupt fields cleared, got at=%v reason=%v", resumed.InterruptedAt, resumed.InterruptReason)
	}
	if resumed.InterruptCount != 1 {
		t.Fatalf("expected interrupt_count preserved (=1), got %d", resumed.InterruptCount)
	}
	if len(resumed.Todos) != 1 || resumed.Todos[0].Status != "pending" {
		t.Fatalf("expected todo reset to pending, got %#v", resumed.Todos)
	}
	if resumed.Todos[0].InterruptedAt != nil || resumed.Todos[0].InterruptReason != nil {
		t.Fatalf("expected todo interrupt fields cleared, got %#v", resumed.Todos[0])
	}

	// 重复 resume 应被拒绝
	if _, appErr := s.ResumeTaskByUser(userID, task.ID); appErr == nil || appErr.Code != "TASK_NOT_INTERRUPTED" {
		t.Fatalf("expected TASK_NOT_INTERRUPTED on second resume, got %#v", appErr)
	}

	events, appErr := s.ListTaskEvents(userID, task.ID)
	if appErr != nil {
		t.Fatalf("list events: %v", appErr)
	}
	resumedEvents := 0
	for _, event := range events {
		if event.EventType == "task_resumed" {
			resumedEvents++
		}
	}
	if resumedEvents != 1 {
		t.Fatalf("expected 1 task_resumed event, got %d", resumedEvents)
	}
}

func TestResumeTaskByUserRestoresPlanningStatus(t *testing.T) {
	s, userID, _, developer, project := seedWorkflowState(t)

	task, appErr := s.CreateTaskPlanning(userID, project.ID, "Need login")
	if appErr != nil {
		t.Fatalf("create planning task: %v", appErr)
	}

	interrupted, appErr := s.InterruptTaskByAgent(developer.NodeID, task.ID, "opencode exited -1")
	if appErr != nil {
		t.Fatalf("interrupt planning task: %v", appErr)
	}
	if interrupted.InterruptedFrom == nil || *interrupted.InterruptedFrom != "planning" {
		t.Fatalf("expected interrupted_from=planning, got %#v", interrupted.InterruptedFrom)
	}

	resumed, appErr := s.ResumeTaskByUser(userID, task.ID)
	if appErr != nil {
		t.Fatalf("resume planning task: %v", appErr)
	}
	if resumed.Status != "planning" {
		t.Fatalf("expected resumed planning status, got %s", resumed.Status)
	}
	if resumed.InterruptedFrom != nil {
		t.Fatalf("expected interrupted_from cleared, got %#v", resumed.InterruptedFrom)
	}
	if len(resumed.Messages) != 1 || resumed.Messages[0].Content != "Need login" {
		t.Fatalf("expected planning messages preserved, got %#v", resumed.Messages)
	}
}

func TestResumeTaskByUserRestoresReviewStatus(t *testing.T) {
	s, userID, pm, developer, project := seedWorkflowState(t)

	task, appErr := s.CreateTaskPlanning(userID, project.ID, "Need login")
	if appErr != nil {
		t.Fatalf("create planning task: %v", appErr)
	}

	task, appErr = s.FinalizePlanByPMNode(pm.NodeID, "msg-plan-ready", TaskPlanReadyInput{
		TaskID:      task.ID,
		Title:       "Implement login",
		Description: "Support email password login",
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
		t.Fatalf("finalize plan: %v", appErr)
	}
	if task.Status != "review" {
		t.Fatalf("expected review status, got %s", task.Status)
	}

	interrupted, appErr := s.InterruptTaskByAgent(pm.NodeID, task.ID, "pm crashed")
	if appErr != nil {
		t.Fatalf("interrupt review task: %v", appErr)
	}
	if interrupted.InterruptedFrom == nil || *interrupted.InterruptedFrom != "review" {
		t.Fatalf("expected interrupted_from=review, got %#v", interrupted.InterruptedFrom)
	}

	resumed, appErr := s.ResumeTaskByUser(userID, task.ID)
	if appErr != nil {
		t.Fatalf("resume review task: %v", appErr)
	}
	if resumed.Status != "review" {
		t.Fatalf("expected resumed review status, got %s", resumed.Status)
	}
	if len(resumed.Todos) != 1 || resumed.Todos[0].Status != "pending" {
		t.Fatalf("expected review todos preserved, got %#v", resumed.Todos)
	}
}

func TestInterruptTaskByAgentTerminalNoStateChange(t *testing.T) {
	s, userID, pm, developer, project := seedWorkflowState(t)

	task, appErr := s.CreateTaskByPMNode(pm.NodeID, TaskCreateInput{
		ProjectID:   project.ID,
		Title:       "Already done",
		Description: "demo",
		Todos: []TaskCreateTodoInput{
			{
				ID:             "todo-1",
				Title:          "Done",
				Description:    "already finished",
				AssigneeNodeID: developer.NodeID,
			},
		},
	})
	if appErr != nil {
		t.Fatalf("create task: %v", appErr)
	}
	if _, appErr := s.CompleteTodoByNodeWithMessageID(developer.NodeID, "msg-1", TodoCompleteInput{
		TaskID: task.ID,
		TodoID: "todo-1",
		Result: model.TodoResult{Summary: "done"},
	}); appErr != nil {
		t.Fatalf("complete todo: %v", appErr)
	}

	result, appErr := s.InterruptTaskByAgent(developer.NodeID, task.ID, "late error")
	if appErr != nil {
		t.Fatalf("interrupt terminal task: %v", appErr)
	}
	if result.Status != "done" {
		t.Fatalf("expected terminal task to remain done, got %s", result.Status)
	}
	if result.InterruptCount != 0 {
		t.Fatalf("expected interrupt_count untouched on terminal task, got %d", result.InterruptCount)
	}

	events, appErr := s.ListTaskEvents(userID, task.ID)
	if appErr != nil {
		t.Fatalf("list events: %v", appErr)
	}
	taskInterruptedCount := 0
	for _, event := range events {
		if event.EventType == "task_interrupted" {
			taskInterruptedCount++
		}
	}
	if taskInterruptedCount != 1 {
		t.Fatalf("expected 1 task_interrupted event for terminal task, got %d", taskInterruptedCount)
	}
}
