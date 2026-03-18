package app

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"trustmesh/backend/internal/config"
	"trustmesh/backend/internal/logger"
	"trustmesh/backend/internal/model"
	"trustmesh/backend/internal/store"
)

func TestHappyPathAuthToConversation(t *testing.T) {
	log, err := logger.New("error")
	if err != nil {
		t.Fatalf("new logger: %v", err)
	}
	defer func() { _ = log.Sync() }()

	application, err := New(config.Config{
		Port:          "0",
		JWTSecret:     "test-secret",
		TokenTTL:      time.Hour,
		LogLevel:      "error",
		AllowAllCORS:  true,
		ReadTimeout:   3 * time.Second,
		WriteTimeout:  3 * time.Second,
		ShutdownGrace: 3 * time.Second,
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
		"email":    "user@example.com",
		"name":     "User",
		"password": "StrongPass123!",
	})
	if registerResp.StatusCode != http.StatusCreated {
		t.Fatalf("register status=%d", registerResp.StatusCode)
	}
	registerData := decodeBody(t, registerResp)
	token := nestedString(registerData, "data", "token")
	if token == "" {
		t.Fatal("empty register token")
	}

	agentResp := doJSON(t, testServer.Client(), "POST", testServer.URL+"/api/v1/agents", token, map[string]any{
		"node_id":      "node-pm-001",
		"name":         "PM Agent",
		"role":         "pm",
		"description":  "PM",
		"capabilities": []string{"plan"},
	})
	if agentResp.StatusCode != http.StatusCreated {
		t.Fatalf("create agent status=%d", agentResp.StatusCode)
	}
	agentData := decodeBody(t, agentResp)
	agentID := nestedString(agentData, "data", "id")
	if agentID == "" {
		t.Fatal("empty agent id")
	}
	application.Store.SyncAgentPresence([]store.AgentPresence{
		{NodeID: "node-pm-001", LastSeenAt: time.Now().UTC()},
	}, time.Now().UTC())

	projectResp := doJSON(t, testServer.Client(), "POST", testServer.URL+"/api/v1/projects", token, map[string]any{
		"name":        "TrustMesh MVP",
		"description": "demo",
		"pm_agent_id": agentID,
	})
	if projectResp.StatusCode != http.StatusCreated {
		t.Fatalf("create project status=%d", projectResp.StatusCode)
	}
	projectData := decodeBody(t, projectResp)
	projectID := nestedString(projectData, "data", "id")
	if projectID == "" {
		t.Fatal("empty project id")
	}

	conversationResp := doJSON(t, testServer.Client(), "POST", testServer.URL+"/api/v1/projects/"+projectID+"/conversations", token, map[string]any{
		"content": "我需要一个登录功能",
	})
	if conversationResp.StatusCode != http.StatusCreated {
		t.Fatalf("create conversation status=%d", conversationResp.StatusCode)
	}
	conversationData := decodeBody(t, conversationResp)
	if nestedString(conversationData, "data", "status") != "active" {
		t.Fatalf("unexpected conversation status: %s", nestedString(conversationData, "data", "status"))
	}

	taskResp := doJSON(t, testServer.Client(), "GET", testServer.URL+"/api/v1/projects/"+projectID+"/tasks", token, nil)
	if taskResp.StatusCode != http.StatusOK {
		t.Fatalf("list tasks status=%d", taskResp.StatusCode)
	}
	taskData := decodeBody(t, taskResp)
	if nestedFloat(taskData, "meta", "count") != 0 {
		t.Fatalf("unexpected task count: %v", nestedFloat(taskData, "meta", "count"))
	}
}

func TestCreateConversationPublishesInitialPMBrief(t *testing.T) {
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
			_, _ = w.Write([]byte(`{"ok":true,"code":"OK","message":"ok","data":{"targetNode":"node-pm-001","messageId":"msg-1"},"ts":1}`))
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
		WriteTimeout:       3 * time.Second,
		ShutdownGrace:      3 * time.Second,
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
		"email":    "brief@example.com",
		"name":     "Brief User",
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
		"name":         "Backend Agent",
		"role":         "developer",
		"description":  "backend",
		"capabilities": []string{"backend", "auth"},
	})
	_ = nestedString(decodeBody(t, devResp), "data", "id")

	reviewerResp := doJSON(t, testServer.Client(), "POST", testServer.URL+"/api/v1/agents", token, map[string]any{
		"node_id":      "node-review-001",
		"name":         "Reviewer Agent",
		"role":         "reviewer",
		"description":  "review",
		"capabilities": []string{"review", "qa"},
	})
	_ = nestedString(decodeBody(t, reviewerResp), "data", "id")

	application.Store.SyncAgentPresence([]store.AgentPresence{
		{NodeID: "node-pm-001", LastSeenAt: time.Now().UTC()},
		{NodeID: "node-dev-001", LastSeenAt: time.Now().UTC()},
		{NodeID: "node-review-001", LastSeenAt: time.Now().UTC()},
	}, time.Now().UTC())

	projectResp := doJSON(t, testServer.Client(), "POST", testServer.URL+"/api/v1/projects", token, map[string]any{
		"name":        "TrustMesh MVP",
		"description": "multi-agent task orchestration",
		"pm_agent_id": pmID,
	})
	projectID := nestedString(decodeBody(t, projectResp), "data", "id")

	conversationResp := doJSON(t, testServer.Client(), "POST", testServer.URL+"/api/v1/projects/"+projectID+"/conversations", token, map[string]any{
		"content": "实现一个带邮箱密码登录和退出能力的认证功能",
	})
	if conversationResp.StatusCode != http.StatusCreated {
		t.Fatalf("create conversation status=%d", conversationResp.StatusCode)
	}

	mu.Lock()
	if len(publishRequests) != 1 {
		t.Fatalf("expected 1 publish request, got %d", len(publishRequests))
	}
	publishReq := publishRequests[0]
	mu.Unlock()

	if publishReq["targetNode"] != "node-pm-001" {
		t.Fatalf("unexpected targetNode: %v", publishReq["targetNode"])
	}
	if publishReq["type"] != "conversation.message" {
		t.Fatalf("unexpected publish type: %v", publishReq["type"])
	}

	rawMessage, ok := publishReq["message"].(string)
	if !ok || rawMessage == "" {
		t.Fatalf("publish message missing: %#v", publishReq["message"])
	}

	var message map[string]any
	if err := json.Unmarshal([]byte(rawMessage), &message); err != nil {
		t.Fatalf("unmarshal publish message: %v", err)
	}

	if message["is_initial_message"] != true {
		t.Fatalf("expected is_initial_message=true, got %#v", message["is_initial_message"])
	}
	if message["user_content"] != "实现一个带邮箱密码登录和退出能力的认证功能" {
		t.Fatalf("unexpected user_content: %#v", message["user_content"])
	}

	content, ok := message["content"].(string)
	if !ok {
		t.Fatalf("content is not string: %#v", message["content"])
	}
	if content == message["user_content"] {
		t.Fatalf("content should differ from user_content for initial message, got: %s", content)
	}
	if !strings.Contains(content, "tm-task-plan") {
		t.Fatalf("content should reference tm-task-plan skill, got: %s", content)
	}

	candidateAgents, ok := message["candidate_agents"].([]any)
	if !ok || len(candidateAgents) != 2 {
		t.Fatalf("unexpected candidate_agents: %#v", message["candidate_agents"])
	}

	pmBrief, ok := message["pm_brief"].(map[string]any)
	if !ok {
		t.Fatalf("pm_brief missing: %#v", message["pm_brief"])
	}
	if pmBrief["objective"] == "" {
		t.Fatalf("pm_brief objective missing: %#v", pmBrief)
	}
	if pmBrief["must_clarify_before_task_create"] != true {
		t.Fatalf("pm_brief must_clarify_before_task_create missing: %#v", pmBrief)
	}
	if pmBrief["must_use_skill"] != "tm-task-plan" {
		t.Fatalf("pm_brief must_use_skill missing: %#v", pmBrief)
	}
}

func TestDispatchTodoPublishesAssignmentToAssignee(t *testing.T) {
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
			_, _ = w.Write([]byte(`{"ok":true,"code":"OK","message":"ok","data":{"targetNode":"node-dev-001","messageId":"msg-1"},"ts":1}`))
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
		WriteTimeout:       3 * time.Second,
		ShutdownGrace:      3 * time.Second,
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
		"email":    "dispatch@example.com",
		"name":     "Dispatch User",
		"password": "StrongPass123!",
	})
	registerData := decodeBody(t, registerResp)
	token := nestedString(registerData, "data", "token")
	userID := nestedString(registerData, "data", "user", "id")

	pm, appErr := application.Store.CreateAgent(userID, "node-pm-001", "PM Agent", "pm", "PM", []string{"plan"})
	if appErr != nil {
		t.Fatalf("create pm: %v", appErr)
	}
	_, appErr = application.Store.CreateAgent(userID, "node-dev-001", "Developer", "developer", "Dev", []string{"backend"})
	if appErr != nil {
		t.Fatalf("create developer: %v", appErr)
	}
	application.Store.SyncAgentPresence([]store.AgentPresence{
		{NodeID: "node-pm-001", LastSeenAt: time.Now().UTC()},
		{NodeID: "node-dev-001", LastSeenAt: time.Now().UTC()},
	}, time.Now().UTC())

	project, appErr := application.Store.CreateProject(userID, "Dispatch Project", "demo", pm.ID)
	if appErr != nil {
		t.Fatalf("create project: %v", appErr)
	}
	conversation, appErr := application.Store.CreateConversation(userID, project.ID, "Need login")
	if appErr != nil {
		t.Fatalf("create conversation: %v", appErr)
	}
	task, appErr := application.Store.CreateTaskByPMNode(pm.NodeID, store.TaskCreateInput{
		ProjectID:      project.ID,
		ConversationID: conversation.ID,
		Title:          "Implement login",
		Description:    "Support email/password login",
		Todos: []store.TaskCreateTodoInput{
			{
				ID:             "todo-1",
				Title:          "Build backend API",
				Description:    "Implement auth endpoints",
				AssigneeNodeID: "node-dev-001",
			},
		},
	})
	if appErr != nil {
		t.Fatalf("create task: %v", appErr)
	}

	dispatchResp := doJSON(t, testServer.Client(), "POST", testServer.URL+"/api/v1/tasks/"+task.ID+"/todos/todo-1/dispatch", token, nil)
	if dispatchResp.StatusCode != http.StatusOK {
		t.Fatalf("dispatch todo status=%d", dispatchResp.StatusCode)
	}

	mu.Lock()
	if len(publishRequests) != 1 {
		t.Fatalf("expected 1 publish request, got %d", len(publishRequests))
	}
	publishReq := publishRequests[0]
	mu.Unlock()

	if publishReq["targetNode"] != "node-dev-001" {
		t.Fatalf("unexpected targetNode: %#v", publishReq["targetNode"])
	}
	if publishReq["type"] != "todo.assigned" {
		t.Fatalf("unexpected publish type: %#v", publishReq["type"])
	}
	if publishReq["sessionKey"] != task.ID {
		t.Fatalf("unexpected session key: %#v", publishReq["sessionKey"])
	}

	rawMessage, ok := publishReq["message"].(string)
	if !ok || rawMessage == "" {
		t.Fatalf("publish message missing: %#v", publishReq["message"])
	}
	var message map[string]any
	if err := json.Unmarshal([]byte(rawMessage), &message); err != nil {
		t.Fatalf("unmarshal publish message: %v", err)
	}
	if message["task_id"] != task.ID || message["todo_id"] != "todo-1" {
		t.Fatalf("unexpected dispatch payload: %#v", message)
	}

	eventsResp := doJSON(t, testServer.Client(), "GET", testServer.URL+"/api/v1/tasks/"+task.ID+"/events", token, nil)
	if eventsResp.StatusCode != http.StatusOK {
		t.Fatalf("list events status=%d", eventsResp.StatusCode)
	}
	eventsData := decodeBody(t, eventsResp)
	items, ok := eventsData["data"].(map[string]any)["items"].([]any)
	if !ok {
		t.Fatalf("unexpected events payload: %#v", eventsData)
	}
	assignedCount := 0
	for _, item := range items {
		event, ok := item.(map[string]any)
		if ok && event["event_type"] == "todo_assigned" {
			assignedCount++
		}
	}
	if assignedCount != 2 {
		t.Fatalf("expected 2 todo_assigned events, got %d", assignedCount)
	}
}

func TestConversationStreamPushesUpdatedSnapshot(t *testing.T) {
	log, err := logger.New("error")
	if err != nil {
		t.Fatalf("new logger: %v", err)
	}
	defer func() { _ = log.Sync() }()

	application, err := New(config.Config{
		Port:          "0",
		JWTSecret:     "test-secret",
		TokenTTL:      time.Hour,
		LogLevel:      "error",
		AllowAllCORS:  true,
		ReadTimeout:   3 * time.Second,
		WriteTimeout:  3 * time.Second,
		ShutdownGrace: 3 * time.Second,
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
		"email":    "stream@example.com",
		"name":     "Stream User",
		"password": "StrongPass123!",
	})
	registerData := decodeBody(t, registerResp)
	token := nestedString(registerData, "data", "token")
	userID := nestedString(registerData, "data", "user", "id")

	pm, appErr := application.Store.CreateAgent(userID, "node-pm-001", "PM Agent", "pm", "PM", []string{"plan"})
	if appErr != nil {
		t.Fatalf("create pm: %v", appErr)
	}
	application.Store.SyncAgentPresence([]store.AgentPresence{
		{NodeID: pm.NodeID, LastSeenAt: time.Now().UTC()},
	}, time.Now().UTC())

	project, appErr := application.Store.CreateProject(userID, "Conversation Stream", "demo", pm.ID)
	if appErr != nil {
		t.Fatalf("create project: %v", appErr)
	}
	conversation, appErr := application.Store.CreateConversation(userID, project.ID, "Need login flow")
	if appErr != nil {
		t.Fatalf("create conversation: %v", appErr)
	}

	streamResp := openSSE(t, testServer.Client(), testServer.URL+"/api/v1/conversations/"+conversation.ID+"/stream", token)
	defer streamResp.Body.Close()
	reader := bufio.NewReader(streamResp.Body)

	eventName, initial := readSSEEvent(t, reader)
	if eventName != "snapshot" {
		t.Fatalf("unexpected initial event: %s", eventName)
	}
	if nestedString(initial, "conversation", "status") != "active" {
		t.Fatalf("unexpected conversation status: %#v", initial)
	}

	if _, appErr := application.Store.AppendPMReplyByNode(pm.NodeID, conversation.ID, "先确认一下登录方式"); appErr != nil {
		t.Fatalf("append pm reply: %v", appErr)
	}

	_, updated := readSSEEvent(t, reader)
	conv, ok := updated["conversation"].(map[string]any)
	if !ok {
		t.Fatalf("unexpected conversation payload: %#v", updated)
	}
	messages, ok := conv["messages"].([]any)
	if !ok || len(messages) != 2 {
		t.Fatalf("unexpected messages: %#v", conv["messages"])
	}
	lastMessage, ok := messages[len(messages)-1].(map[string]any)
	if !ok || lastMessage["content"] != "先确认一下登录方式" {
		t.Fatalf("unexpected last message: %#v", lastMessage)
	}
}

func TestTaskStreamPushesUpdatedSnapshot(t *testing.T) {
	log, err := logger.New("error")
	if err != nil {
		t.Fatalf("new logger: %v", err)
	}
	defer func() { _ = log.Sync() }()

	application, err := New(config.Config{
		Port:          "0",
		JWTSecret:     "test-secret",
		TokenTTL:      time.Hour,
		LogLevel:      "error",
		AllowAllCORS:  true,
		ReadTimeout:   3 * time.Second,
		WriteTimeout:  3 * time.Second,
		ShutdownGrace: 3 * time.Second,
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
		"email":    "task-stream@example.com",
		"name":     "Task Stream User",
		"password": "StrongPass123!",
	})
	registerData := decodeBody(t, registerResp)
	token := nestedString(registerData, "data", "token")
	userID := nestedString(registerData, "data", "user", "id")

	pm, appErr := application.Store.CreateAgent(userID, "node-pm-001", "PM Agent", "pm", "PM", []string{"plan"})
	if appErr != nil {
		t.Fatalf("create pm: %v", appErr)
	}
	dev, appErr := application.Store.CreateAgent(userID, "node-dev-001", "Developer", "developer", "Dev", []string{"backend"})
	if appErr != nil {
		t.Fatalf("create developer: %v", appErr)
	}
	application.Store.SyncAgentPresence([]store.AgentPresence{
		{NodeID: pm.NodeID, LastSeenAt: time.Now().UTC()},
		{NodeID: dev.NodeID, LastSeenAt: time.Now().UTC()},
	}, time.Now().UTC())

	project, appErr := application.Store.CreateProject(userID, "Task Stream", "demo", pm.ID)
	if appErr != nil {
		t.Fatalf("create project: %v", appErr)
	}
	conversation, appErr := application.Store.CreateConversation(userID, project.ID, "Need login flow")
	if appErr != nil {
		t.Fatalf("create conversation: %v", appErr)
	}
	task, appErr := application.Store.CreateTaskByPMNode(pm.NodeID, store.TaskCreateInput{
		ProjectID:      project.ID,
		ConversationID: conversation.ID,
		Title:          "Implement login",
		Description:    "Support email/password login",
		Todos: []store.TaskCreateTodoInput{
			{
				ID:             "todo-1",
				Title:          "Build backend API",
				Description:    "Implement auth endpoints",
				AssigneeNodeID: dev.NodeID,
			},
		},
	})
	if appErr != nil {
		t.Fatalf("create task: %v", appErr)
	}

	streamResp := openSSE(t, testServer.Client(), testServer.URL+"/api/v1/tasks/"+task.ID+"/stream", token)
	defer streamResp.Body.Close()
	reader := bufio.NewReader(streamResp.Body)

	eventName, initial := readSSEEvent(t, reader)
	if eventName != "snapshot" {
		t.Fatalf("unexpected initial event: %s", eventName)
	}
	if nestedString(initial, "task", "status") != "pending" {
		t.Fatalf("unexpected initial task status: %#v", initial)
	}

	if _, appErr := application.Store.UpdateTodoProgressByNode(dev.NodeID, store.TodoProgressInput{
		TaskID:  task.ID,
		TodoID:  "todo-1",
		Message: "Started implementing auth endpoints",
	}); appErr != nil {
		t.Fatalf("update todo progress: %v", appErr)
	}

	_, updated := readSSEEvent(t, reader)
	if nestedString(updated, "task", "status") != "in_progress" {
		t.Fatalf("unexpected updated task status: %#v", updated)
	}
	events, ok := updated["events"].([]any)
	if !ok || len(events) < 3 {
		t.Fatalf("unexpected task events: %#v", updated["events"])
	}
	lastEvent, ok := events[len(events)-1].(map[string]any)
	if !ok || lastEvent["event_type"] != "task_status_changed" {
		t.Fatalf("unexpected last event: %#v", lastEvent)
	}
}

func TestGetTaskArtifactTransfer(t *testing.T) {
	log, err := logger.New("error")
	if err != nil {
		t.Fatalf("new logger: %v", err)
	}
	defer func() { _ = log.Sync() }()

	clawServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/transfer/tf_report_123":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"ok":true,"code":"transfer.ok","message":"ok","data":{"transfer":{"transfer_id":"tf_report_123","download_url":"https://files.example.com/tf_report_123","size":2048}},"ts":1}`))
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
		WriteTimeout:       3 * time.Second,
		ShutdownGrace:      3 * time.Second,
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
		"email":    "transfer@example.com",
		"name":     "Transfer User",
		"password": "StrongPass123!",
	})
	registerData := decodeBody(t, registerResp)
	token := nestedString(registerData, "data", "token")
	userID := nestedString(registerData, "data", "user", "id")

	pm, appErr := application.Store.CreateAgent(userID, "node-pm-001", "PM Agent", "pm", "pm", []string{"plan"})
	if appErr != nil {
		t.Fatalf("create pm: %v", appErr)
	}
	dev, appErr := application.Store.CreateAgent(userID, "node-dev-001", "Developer", "developer", "dev", []string{"backend"})
	if appErr != nil {
		t.Fatalf("create dev: %v", appErr)
	}
	application.Store.SyncAgentPresence([]store.AgentPresence{
		{NodeID: pm.NodeID, LastSeenAt: time.Now().UTC()},
		{NodeID: dev.NodeID, LastSeenAt: time.Now().UTC()},
	}, time.Now().UTC())

	project, appErr := application.Store.CreateProject(userID, "Transfers", "demo", pm.ID)
	if appErr != nil {
		t.Fatalf("create project: %v", appErr)
	}
	conversation, appErr := application.Store.CreateConversation(userID, project.ID, "Need file")
	if appErr != nil {
		t.Fatalf("create conversation: %v", appErr)
	}
	task, appErr := application.Store.CreateTaskByPMNode(pm.NodeID, store.TaskCreateInput{
		ProjectID:      project.ID,
		ConversationID: conversation.ID,
		Title:          "Deliver report",
		Description:    "Upload final report",
		Todos: []store.TaskCreateTodoInput{
			{
				ID:             "todo-1",
				Title:          "Upload report",
				Description:    "Send report PDF",
				AssigneeNodeID: dev.NodeID,
			},
		},
	})
	if appErr != nil {
		t.Fatalf("create task: %v", appErr)
	}
	task, appErr = application.Store.CompleteTodoByNode(dev.NodeID, store.TodoCompleteInput{
		TaskID: task.ID,
		TodoID: "todo-1",
		Result: todoResultWithTransfer(),
	})
	if appErr != nil {
		t.Fatalf("complete todo: %v", appErr)
	}

	if len(task.Artifacts) != 1 {
		t.Fatalf("expected 1 artifact, got %d", len(task.Artifacts))
	}
	resp := doJSON(t, testServer.Client(), "GET", testServer.URL+"/api/v1/tasks/"+task.ID+"/artifacts/"+task.Artifacts[0].ID+"/transfer", token, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get transfer status=%d", resp.StatusCode)
	}
	data := decodeBody(t, resp)
	if nestedString(data, "data", "transfer_id") != "tf_report_123" {
		t.Fatalf("unexpected transfer response: %#v", data)
	}
	if nestedString(data, "data", "download_url") != "https://files.example.com/tf_report_123" {
		t.Fatalf("unexpected download_url: %#v", data)
	}
}

func TestGetTaskArtifactContent(t *testing.T) {
	log, err := logger.New("error")
	if err != nil {
		t.Fatalf("new logger: %v", err)
	}
	defer func() { _ = log.Sync() }()

	application, err := New(config.Config{
		Port:          "0",
		JWTSecret:     "test-secret",
		TokenTTL:      time.Hour,
		LogLevel:      "error",
		AllowAllCORS:  true,
		ReadTimeout:   3 * time.Second,
		WriteTimeout:  3 * time.Second,
		ShutdownGrace: 3 * time.Second,
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
		"email":    "artifact-content@example.com",
		"name":     "Artifact Content User",
		"password": "StrongPass123!",
	})
	registerData := decodeBody(t, registerResp)
	token := nestedString(registerData, "data", "token")
	userID := nestedString(registerData, "data", "user", "id")

	pm, appErr := application.Store.CreateAgent(userID, "node-pm-001", "PM Agent", "pm", "pm", []string{"plan"})
	if appErr != nil {
		t.Fatalf("create pm: %v", appErr)
	}
	dev, appErr := application.Store.CreateAgent(userID, "node-dev-001", "Developer", "developer", "dev", []string{"backend"})
	if appErr != nil {
		t.Fatalf("create dev: %v", appErr)
	}
	application.Store.SyncAgentPresence([]store.AgentPresence{
		{NodeID: pm.NodeID, LastSeenAt: time.Now().UTC()},
		{NodeID: dev.NodeID, LastSeenAt: time.Now().UTC()},
	}, time.Now().UTC())

	project, appErr := application.Store.CreateProject(userID, "Artifacts", "demo", pm.ID)
	if appErr != nil {
		t.Fatalf("create project: %v", appErr)
	}
	conversation, appErr := application.Store.CreateConversation(userID, project.ID, "Need content")
	if appErr != nil {
		t.Fatalf("create conversation: %v", appErr)
	}
	task, appErr := application.Store.CreateTaskByPMNode(pm.NodeID, store.TaskCreateInput{
		ProjectID:      project.ID,
		ConversationID: conversation.ID,
		Title:          "Deliver document",
		Description:    "Upload markdown guide",
		Todos: []store.TaskCreateTodoInput{
			{
				ID:             "todo-1",
				Title:          "Write guide",
				Description:    "Create markdown file",
				AssigneeNodeID: dev.NodeID,
			},
		},
	})
	if appErr != nil {
		t.Fatalf("create task: %v", appErr)
	}

	tmpFile, err := os.CreateTemp(t.TempDir(), "guide-*.md")
	if err != nil {
		t.Fatalf("create temp file: %v", err)
	}
	content := "# Git guide\n\nhello"
	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatalf("write temp file: %v", err)
	}
	if err := tmpFile.Close(); err != nil {
		t.Fatalf("close temp file: %v", err)
	}

	task, appErr = application.Store.CompleteTodoByNode(dev.NodeID, store.TodoCompleteInput{
		TaskID: task.ID,
		TodoID: "todo-1",
		Result: model.TodoResult{
			Summary: "Guide uploaded",
			Output:  "Uploaded markdown guide",
			ArtifactRefs: []model.TodoResultArtifactRef{
				{
					ArtifactID: "tf_markdown_1",
					Kind:       "file",
					Label:      "Git guide",
				},
			},
			Metadata: map[string]any{
				"transfers": []any{
					map[string]any{
						"transfer_id": "tf_markdown_1",
						"fileName":    "git-guide.md",
						"localPath":   tmpFile.Name(),
						"mimeType":    "text/markdown",
					},
				},
			},
		},
	})
	if appErr != nil {
		t.Fatalf("complete todo: %v", appErr)
	}

	resp := doJSON(t, testServer.Client(), "GET", testServer.URL+"/api/v1/tasks/"+task.ID+"/artifacts/"+task.Artifacts[0].ID+"/content", token, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get content status=%d", resp.StatusCode)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read content body: %v", err)
	}
	if string(body) != content {
		t.Fatalf("unexpected content body: %s", string(body))
	}
	if !strings.Contains(resp.Header.Get("Content-Disposition"), `filename="git-guide.md"`) {
		t.Fatalf("unexpected content disposition: %s", resp.Header.Get("Content-Disposition"))
	}
}

func todoResultWithTransfer() model.TodoResult {
	return model.TodoResult{
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
					"size":        2048,
					"checksum":    "sha256:abc123",
				},
			},
		},
	}
}

func doJSON(t *testing.T, client *http.Client, method, url, token string, payload any) *http.Response {
	t.Helper()
	var body bytes.Buffer
	if payload != nil {
		if err := json.NewEncoder(&body).Encode(payload); err != nil {
			t.Fatalf("encode payload: %v", err)
		}
	}
	req, err := http.NewRequest(method, url, &body)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	return resp
}

func openSSE(t *testing.T, client *http.Client, url, token string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		t.Fatalf("new sse request: %v", err)
	}
	req.Header.Set("Accept", "text/event-stream")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("open sse request failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("open sse status=%d", resp.StatusCode)
	}
	return resp
}

func readSSEEvent(t *testing.T, reader *bufio.Reader) (string, map[string]any) {
	t.Helper()

	type result struct {
		event string
		data  map[string]any
		err   error
	}

	done := make(chan result, 1)
	go func() {
		eventName := "message"
		dataLines := make([]string, 0, 4)
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				done <- result{err: err}
				return
			}
			line = strings.TrimRight(line, "\r\n")
			if line == "" {
				break
			}
			if strings.HasPrefix(line, "event:") {
				eventName = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
				continue
			}
			if strings.HasPrefix(line, "data:") {
				dataLines = append(dataLines, strings.TrimSpace(strings.TrimPrefix(line, "data:")))
			}
		}

		var payload map[string]any
		if err := json.Unmarshal([]byte(strings.Join(dataLines, "\n")), &payload); err != nil {
			done <- result{err: err}
			return
		}
		done <- result{event: eventName, data: payload}
	}()

	select {
	case res := <-done:
		if res.err != nil {
			t.Fatalf("read sse event: %v", res.err)
		}
		return res.event, res.data
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for sse event")
		return "", nil
	}
}

func decodeBody(t *testing.T, resp *http.Response) map[string]any {
	t.Helper()
	defer resp.Body.Close()
	var out map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	return out
}

func nestedString(m map[string]any, keys ...string) string {
	cur := any(m)
	for _, key := range keys {
		obj, ok := cur.(map[string]any)
		if !ok {
			return ""
		}
		cur, ok = obj[key]
		if !ok {
			return ""
		}
	}
	v, ok := cur.(string)
	if !ok {
		return ""
	}
	return v
}

func nestedFloat(m map[string]any, keys ...string) float64 {
	cur := any(m)
	for _, key := range keys {
		obj, ok := cur.(map[string]any)
		if !ok {
			return -1
		}
		cur, ok = obj[key]
		if !ok {
			return -1
		}
	}
	v, ok := cur.(float64)
	if !ok {
		return -1
	}
	return v
}

func nestedMap(m map[string]any, keys ...string) map[string]any {
	cur := any(m)
	for _, key := range keys {
		obj, ok := cur.(map[string]any)
		if !ok {
			return nil
		}
		cur, ok = obj[key]
		if !ok {
			return nil
		}
	}
	v, _ := cur.(map[string]any)
	return v
}
