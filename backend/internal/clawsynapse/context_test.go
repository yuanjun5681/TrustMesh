package clawsynapse

import (
	"testing"

	"trustmesh/backend/internal/model"
)

func makeTodo(id string, order int, status, nodeID, name string, result model.TodoResult) model.Todo {
	return model.Todo{
		ID:          id,
		Order:       order,
		Title:       "Todo " + id,
		Description: "Description " + id,
		Status:      status,
		Assignee:    model.TodoAssignee{AgentID: "agent-" + nodeID, Name: name, NodeID: nodeID},
		Result:      result,
	}
}

func makeTask(todos []model.Todo) *model.TaskDetail {
	return &model.TaskDetail{
		ID:          "task-1",
		Title:       "Test Task",
		Description: "Test task description",
		Todos:       todos,
		PMAgent:     model.PMAgentSummary{NodeID: "pm-node"},
	}
}

func TestAgentHasPriorTodoInTask(t *testing.T) {
	tests := []struct {
		name     string
		todos    []model.Todo
		current  string // todo ID
		expected bool
	}{
		{
			name: "first todo for agent",
			todos: []model.Todo{
				makeTodo("t1", 1, "pending", "node-A", "AgentA", model.TodoResult{}),
			},
			current:  "t1",
			expected: false,
		},
		{
			name: "agent has prior completed todo",
			todos: []model.Todo{
				makeTodo("t1", 1, "done", "node-A", "AgentA", model.TodoResult{Summary: "done"}),
				makeTodo("t2", 2, "done", "node-B", "AgentB", model.TodoResult{Summary: "done"}),
				makeTodo("t3", 3, "pending", "node-A", "AgentA", model.TodoResult{}),
			},
			current:  "t3",
			expected: true,
		},
		{
			name: "agent has prior failed todo",
			todos: []model.Todo{
				makeTodo("t1", 1, "failed", "node-A", "AgentA", model.TodoResult{}),
				makeTodo("t2", 2, "pending", "node-A", "AgentA", model.TodoResult{}),
			},
			current:  "t2",
			expected: true,
		},
		{
			name: "different agent has prior todo",
			todos: []model.Todo{
				makeTodo("t1", 1, "done", "node-B", "AgentB", model.TodoResult{Summary: "done"}),
				makeTodo("t2", 2, "pending", "node-A", "AgentA", model.TodoResult{}),
			},
			current:  "t2",
			expected: false,
		},
		{
			name: "agent has later todo only",
			todos: []model.Todo{
				makeTodo("t1", 1, "pending", "node-A", "AgentA", model.TodoResult{}),
				makeTodo("t2", 2, "done", "node-A", "AgentA", model.TodoResult{Summary: "done"}),
			},
			current:  "t1",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := makeTask(tt.todos)
			var current *model.Todo
			for i := range task.Todos {
				if task.Todos[i].ID == tt.current {
					current = &task.Todos[i]
					break
				}
			}
			if current == nil {
				t.Fatalf("current todo %q not found", tt.current)
			}
			got := agentHasPriorTodoInTask(task, current)
			if got != tt.expected {
				t.Errorf("agentHasPriorTodoInTask() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestBuildTaskContext(t *testing.T) {
	task := makeTask([]model.Todo{
		makeTodo("t1", 1, "done", "node-A", "AgentA", model.TodoResult{}),
		makeTodo("t2", 2, "pending", "node-B", "AgentB", model.TodoResult{}),
		makeTodo("t3", 3, "pending", "node-A", "AgentA", model.TodoResult{}),
	})

	ctx := buildTaskContext(task, "t2")

	if ctx.Title != "Test Task" {
		t.Errorf("title = %q, want %q", ctx.Title, "Test Task")
	}
	if len(ctx.Todos) != 3 {
		t.Fatalf("todos count = %d, want 3", len(ctx.Todos))
	}
	if !ctx.Todos[1].IsCurrent {
		t.Error("todos[1] should be current")
	}
	if ctx.Todos[0].IsCurrent || ctx.Todos[2].IsCurrent {
		t.Error("only todos[1] should be current")
	}
}

func TestBuildAllPriorResults(t *testing.T) {
	task := makeTask([]model.Todo{
		makeTodo("t1", 1, "done", "node-A", "AgentA", model.TodoResult{
			Summary: "Result 1",
			Output:  "Output 1",
		}),
		makeTodo("t2", 2, "done", "node-B", "AgentB", model.TodoResult{
			Summary: "Result 2",
			Output:  "Output 2",
		}),
		makeTodo("t3", 3, "pending", "node-C", "AgentC", model.TodoResult{}),
	})

	current := &task.Todos[2] // t3
	results := buildAllPriorResults(task, current)

	if len(results) != 2 {
		t.Fatalf("results count = %d, want 2", len(results))
	}
	if results[0].TodoID != "t1" || results[0].Summary != "Result 1" {
		t.Errorf("results[0] = %+v", results[0])
	}
	if results[1].TodoID != "t2" || results[1].Summary != "Result 2" {
		t.Errorf("results[1] = %+v", results[1])
	}
}

func TestBuildCrossAgentPriorResults(t *testing.T) {
	task := makeTask([]model.Todo{
		makeTodo("t1", 1, "done", "node-A", "AgentA", model.TodoResult{
			Summary: "Result A1",
			Output:  "Output A1",
		}),
		makeTodo("t2", 2, "done", "node-B", "AgentB", model.TodoResult{
			Summary: "Result B",
			Output:  "Output B",
		}),
		makeTodo("t3", 3, "pending", "node-A", "AgentA", model.TodoResult{}),
	})

	current := &task.Todos[2] // t3 assigned to node-A
	results := buildCrossAgentPriorResults(task, current)

	// Should only include t2 (node-B), not t1 (node-A, same agent)
	if len(results) != 1 {
		t.Fatalf("results count = %d, want 1", len(results))
	}
	if results[0].TodoID != "t2" {
		t.Errorf("expected t2, got %s", results[0].TodoID)
	}
	if results[0].Summary != "Result B" {
		t.Errorf("summary = %q, want %q", results[0].Summary, "Result B")
	}
}

func TestBuildTodoAssignedPayload_FirstTimeAgent(t *testing.T) {
	task := makeTask([]model.Todo{
		makeTodo("t1", 1, "done", "node-A", "AgentA", model.TodoResult{Summary: "Done A"}),
		makeTodo("t2", 2, "pending", "node-B", "AgentB", model.TodoResult{}),
	})

	h := &WebhookHandler{}
	payload := h.buildTodoAssignedPayload(task, &task.Todos[1])

	if payload.TaskContext == nil {
		t.Fatal("expected task_context for first-time agent")
	}
	if payload.TaskContext.Title != "Test Task" {
		t.Errorf("task_context.title = %q", payload.TaskContext.Title)
	}
	if len(payload.PriorResults) != 1 || payload.PriorResults[0].TodoID != "t1" {
		t.Errorf("expected prior result for t1, got %+v", payload.PriorResults)
	}
}

func TestBuildTodoAssignedPayload_ReturningAgent(t *testing.T) {
	task := makeTask([]model.Todo{
		makeTodo("t1", 1, "done", "node-A", "AgentA", model.TodoResult{Summary: "Done A1"}),
		makeTodo("t2", 2, "done", "node-B", "AgentB", model.TodoResult{Summary: "Done B"}),
		makeTodo("t3", 3, "pending", "node-A", "AgentA", model.TodoResult{}),
	})

	h := &WebhookHandler{}
	payload := h.buildTodoAssignedPayload(task, &task.Todos[2])

	// Returning agent: no task_context (already in session)
	if payload.TaskContext != nil {
		t.Error("expected nil task_context for returning agent")
	}
	// Only cross-agent results (t2 from node-B), not t1 (own work)
	if len(payload.PriorResults) != 1 {
		t.Fatalf("expected 1 cross-agent result, got %d", len(payload.PriorResults))
	}
	if payload.PriorResults[0].TodoID != "t2" {
		t.Errorf("expected t2, got %s", payload.PriorResults[0].TodoID)
	}
}

func TestBuildTodoAssignedPayload_NoPriorResults(t *testing.T) {
	task := makeTask([]model.Todo{
		makeTodo("t1", 1, "pending", "node-A", "AgentA", model.TodoResult{}),
	})

	h := &WebhookHandler{}
	payload := h.buildTodoAssignedPayload(task, &task.Todos[0])

	if payload.TaskContext == nil {
		t.Fatal("expected task_context for first todo")
	}
	if len(payload.PriorResults) != 0 {
		t.Errorf("expected no prior results, got %d", len(payload.PriorResults))
	}
}

func TestBuildPriorResult(t *testing.T) {
	todo := &model.Todo{
		ID:     "t1",
		Title:  "Design API",
		Status: "done",
		Result: model.TodoResult{
			Summary: "API designed",
			Output:  "Detailed design doc",
		},
	}

	r := buildPriorResult(todo)

	if r.Summary != "API designed" || r.Output != "Detailed design doc" {
		t.Errorf("unexpected result: %+v", r)
	}
	if r.TodoID != "t1" || r.Title != "Design API" {
		t.Errorf("unexpected identity: %+v", r)
	}
}
