package app

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
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

	var (
		mu              sync.Mutex
		publishRequests []map[string]any
	)

	clawServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/peers":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(clawsynapsePeersResponse(t, "node-pm-001", "node-dev-001")))
		case "/v1/publish":
			defer r.Body.Close()
			var payload map[string]any
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("decode publish payload: %v", err)
			}
			mu.Lock()
			publishRequests = append(publishRequests, payload)
			mu.Unlock()
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"ok":true,"code":"msg.published","message":"ok","data":{"targetNode":"node-dev-001","messageId":"msg-1"},"ts":1}`))
		case "/v1/transfer/tf_login_guide":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"ok":true,"code":"transfer.detail","message":"ok","data":{"transfer":{"transferId":"tf_login_guide","fileName":"login-guide.md","localPath":"/tmp/login-guide.md","mimeType":"text/markdown","fileSize":321,"status":"completed"}},"ts":1}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer clawServer.Close()

	application, err := New(config.Config{
		Port:               "0",
		JWTSecret:          "test-secret",
		TokenTTL:           time.Hour,
		LogLevel:           "error",
		AllowAllCORS:       true,
		ReadTimeout:        3 * time.Second,
		ShutdownGrace:      3 * time.Second,
		ClawSynapseNodeID:  "trustmesh-server",
		ClawSynapseAPIURL:  clawServer.URL,
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

	mu.Lock()
	initialPublishCount := len(publishRequests)
	mu.Unlock()

	taskCreateMessage, err := json.Marshal(map[string]any{
		"project_id":      projectID,
		"conversation_id": conversationID,
		"title":           "Implement login",
		"description":     "Support email and password login",
		"todos": []map[string]any{
			{
				"id":               "todo-1",
				"order":            1,
				"title":            "Build backend API",
				"description":      "Implement auth endpoints",
				"assignee_node_id": "node-dev-001",
			},
			{
				"id":               "todo-2",
				"order":            2,
				"title":            "Build frontend UI",
				"description":      "Implement login form after backend API is ready",
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

	mu.Lock()
	if len(publishRequests) != initialPublishCount+2 {
		t.Fatalf("expected %d publish requests after task.create, got %d", initialPublishCount+2, len(publishRequests))
	}
	taskCreatedReq := publishRequests[initialPublishCount]
	firstTodoAssignedReq := publishRequests[initialPublishCount+1]
	mu.Unlock()

	if taskCreatedReq["type"] != "task.created" || taskCreatedReq["targetNode"] != "node-pm-001" {
		t.Fatalf("unexpected task.created publish request: %#v", taskCreatedReq)
	}
	if firstTodoAssignedReq["type"] != "todo.assigned" || firstTodoAssignedReq["targetNode"] != "node-dev-001" {
		t.Fatalf("unexpected todo.assigned publish request: %#v", firstTodoAssignedReq)
	}

	todoCompleteMessage, err := json.Marshal(map[string]any{
		"task_id": taskID,
		"todo_id": "todo-1",
		"result": map[string]any{
			"summary": "done",
			"output":  "implemented login endpoints",
			"artifact_refs": []map[string]any{
				{
					"artifact_id": "tf_login_guide",
					"kind":        "file",
					"label":       "Login guide",
				},
			},
			"metadata": map[string]any{
				"duration_ms": 1200,
				"transfers": []map[string]any{
					{
						"transfer_id": "tf_login_guide",
					},
				},
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

	mu.Lock()
	if len(publishRequests) != initialPublishCount+5 {
		t.Fatalf("expected %d publish requests after todo.complete, got %d", initialPublishCount+5, len(publishRequests))
	}
	taskStatusReq := publishRequests[initialPublishCount+2]
	todoStatusReq := publishRequests[initialPublishCount+3]
	secondTodoAssignedReq := publishRequests[initialPublishCount+4]
	mu.Unlock()

	if taskStatusReq["type"] != "task.status_changed" || taskStatusReq["targetNode"] != "node-pm-001" {
		t.Fatalf("unexpected task.status_changed publish request: %#v", taskStatusReq)
	}
	if todoStatusReq["type"] != "todo.status_changed" || todoStatusReq["targetNode"] != "node-pm-001" {
		t.Fatalf("unexpected todo.status_changed publish request: %#v", todoStatusReq)
	}
	if secondTodoAssignedReq["type"] != "todo.assigned" || secondTodoAssignedReq["targetNode"] != "node-dev-001" {
		t.Fatalf("unexpected sequential todo.assigned publish request: %#v", secondTodoAssignedReq)
	}

	rawTaskStatus, ok := taskStatusReq["message"].(string)
	if !ok || rawTaskStatus == "" {
		t.Fatalf("task.status_changed message missing: %#v", taskStatusReq["message"])
	}
	var taskStatusMessage map[string]any
	if err := json.Unmarshal([]byte(rawTaskStatus), &taskStatusMessage); err != nil {
		t.Fatalf("unmarshal task.status_changed message: %v", err)
	}
	if taskStatusMessage["actor_node_id"] != "node-dev-001" || taskStatusMessage["cause"] != "todo.complete" {
		t.Fatalf("unexpected task.status_changed payload: %#v", taskStatusMessage)
	}
	if taskStatusMessage["version"] == nil {
		t.Fatalf("task.status_changed version missing: %#v", taskStatusMessage)
	}

	rawTodoStatus, ok := todoStatusReq["message"].(string)
	if !ok || rawTodoStatus == "" {
		t.Fatalf("todo.status_changed message missing: %#v", todoStatusReq["message"])
	}
	var todoStatusMessage map[string]any
	if err := json.Unmarshal([]byte(rawTodoStatus), &todoStatusMessage); err != nil {
		t.Fatalf("unmarshal todo.status_changed message: %v", err)
	}
	if todoStatusMessage["actor_node_id"] != "node-dev-001" || todoStatusMessage["cause"] != "todo.complete" {
		t.Fatalf("unexpected todo.status_changed payload: %#v", todoStatusMessage)
	}
	if todoStatusMessage["status"] != "done" {
		t.Fatalf("unexpected todo.status_changed status: %#v", todoStatusMessage)
	}

	taskResp := doJSON(t, testServer.Client(), "GET", testServer.URL+"/api/v1/tasks/"+taskID, token, nil)
	taskData := decodeBody(t, taskResp)
	if nestedString(taskData, "data", "status") != "in_progress" {
		t.Fatalf("unexpected task status: %s", nestedString(taskData, "data", "status"))
	}
	artifacts, ok := nestedMap(taskData, "data")["artifacts"].([]any)
	if !ok || len(artifacts) != 1 {
		t.Fatalf("unexpected artifacts: %#v", nestedMap(taskData, "data")["artifacts"])
	}
	artifact, ok := artifacts[0].(map[string]any)
	if !ok {
		t.Fatalf("artifact is not object: %#v", artifacts[0])
	}
	metadata, ok := artifact["metadata"].(map[string]any)
	if !ok {
		t.Fatalf("artifact metadata is not object: %#v", artifact["metadata"])
	}
	if metadata["file_name"] != "login-guide.md" {
		t.Fatalf("expected file_name to be persisted, got %#v", metadata["file_name"])
	}
	if metadata["local_path"] != "/tmp/login-guide.md" {
		t.Fatalf("expected local_path to be persisted, got %#v", metadata["local_path"])
	}
}
