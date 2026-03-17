package app

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"trustmesh/backend/internal/config"
	"trustmesh/backend/internal/logger"
	"trustmesh/backend/internal/store"
)

func TestWebhookTaskLifecycle(t *testing.T) {
	log, err := logger.New("error")
	if err != nil {
		t.Fatalf("new logger: %v", err)
	}
	defer func() { _ = log.Sync() }()

	application, err := New(config.Config{
		Port:               "0",
		JWTSecret:          "test-secret",
		TokenTTL:           time.Hour,
		LogLevel:           "error",
		AllowAllCORS:       true,
		ReadTimeout:        3 * time.Second,
		WriteTimeout:       3 * time.Second,
		ShutdownGrace:      3 * time.Second,
		ClawSynapseNodeID:  "trustmesh-server",
		ClawSynapseAPIURL:  "",
		ClawSynapseTimeout: time.Second,
	}, log)
	if err != nil {
		t.Fatalf("new app: %v", err)
	}
	defer func() {
		if closeErr := application.Close(); closeErr != nil {
			t.Fatalf("close app: %v", closeErr)
		}
	}()

	testServer := httptest.NewServer(application.Engine)
	defer testServer.Close()

	registerResp := doJSON(t, testServer.Client(), "POST", testServer.URL+"/api/v1/auth/register", "", map[string]any{
		"email":    "webhook@example.com",
		"name":     "Webhook User",
		"password": "StrongPass123!",
	})
	token := nestedString(decodeBody(t, registerResp), "data", "token")

	pmResp := doJSON(t, testServer.Client(), "POST", testServer.URL+"/api/v1/agents", token, map[string]any{
		"node_id":      "node-pm-001",
		"name":         "PM Agent",
		"role":         "pm",
		"description":  "PM",
		"capabilities": []string{"plan"},
	})
	pmID := nestedString(decodeBody(t, pmResp), "data", "id")

	devResp := doJSON(t, testServer.Client(), "POST", testServer.URL+"/api/v1/agents", token, map[string]any{
		"node_id":      "node-dev-001",
		"name":         "Developer",
		"role":         "developer",
		"description":  "Dev",
		"capabilities": []string{"backend"},
	})
	_ = nestedString(decodeBody(t, devResp), "data", "id")

	application.Store.SyncAgentPresence([]store.AgentPresence{
		{NodeID: "node-pm-001", LastSeenAt: time.Now().UTC()},
		{NodeID: "node-dev-001", LastSeenAt: time.Now().UTC()},
	}, time.Now().UTC())

	projectResp := doJSON(t, testServer.Client(), "POST", testServer.URL+"/api/v1/projects", token, map[string]any{
		"name":        "Webhook Project",
		"description": "demo",
		"pm_agent_id": pmID,
	})
	projectID := nestedString(decodeBody(t, projectResp), "data", "id")

	conversationResp := doJSON(t, testServer.Client(), "POST", testServer.URL+"/api/v1/projects/"+projectID+"/conversations", token, map[string]any{
		"content": "Build login flow",
	})
	conversationID := nestedString(decodeBody(t, conversationResp), "data", "id")

	taskCreateMessage, err := json.Marshal(map[string]any{
		"project_id":      projectID,
		"conversation_id": conversationID,
		"title":           "Implement login",
		"description":     "Support email and password login",
		"todos": []map[string]any{
			{
				"id":               "todo-1",
				"title":            "Build backend API",
				"description":      "Implement auth endpoints",
				"assignee_node_id": "node-dev-001",
			},
		},
	})
	if err != nil {
		t.Fatalf("marshal task.create: %v", err)
	}

	webhookResp := doJSON(t, testServer.Client(), "POST", testServer.URL+"/webhook/clawsynapse", "", map[string]any{
		"nodeId":  "trustmesh-server",
		"type":    "task.create",
		"from":    "node-pm-001",
		"message": string(taskCreateMessage),
		"metadata": map[string]any{
			"messageId": "msg-task-create-1",
		},
	})
	if webhookResp.StatusCode != http.StatusOK {
		t.Fatalf("task.create webhook status=%d", webhookResp.StatusCode)
	}
	taskID := nestedString(decodeBody(t, webhookResp), "data", "id")
	if taskID == "" {
		t.Fatal("empty task id")
	}

	todoCompleteMessage, err := json.Marshal(map[string]any{
		"task_id": taskID,
		"todo_id": "todo-1",
		"result": map[string]any{
			"summary":       "done",
			"output":        "implemented login endpoints",
			"artifact_refs": []map[string]any{},
			"metadata": map[string]any{
				"duration_ms": 1200,
			},
		},
	})
	if err != nil {
		t.Fatalf("marshal todo.complete: %v", err)
	}

	completeResp := doJSON(t, testServer.Client(), "POST", testServer.URL+"/webhook/clawsynapse", "", map[string]any{
		"nodeId":  "trustmesh-server",
		"type":    "todo.complete",
		"from":    "node-dev-001",
		"message": string(todoCompleteMessage),
		"metadata": map[string]any{
			"messageId": "msg-todo-complete-1",
		},
	})
	if completeResp.StatusCode != http.StatusOK {
		t.Fatalf("todo.complete webhook status=%d", completeResp.StatusCode)
	}

	taskResp := doJSON(t, testServer.Client(), "GET", testServer.URL+"/api/v1/tasks/"+taskID, token, nil)
	taskData := decodeBody(t, taskResp)
	if nestedString(taskData, "data", "status") != "done" {
		t.Fatalf("unexpected task status: %s", nestedString(taskData, "data", "status"))
	}
}
