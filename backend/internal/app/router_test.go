package app

import (
	"bytes"
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
	if content != "实现一个带邮箱密码登录和退出能力的认证功能" {
		t.Fatalf("content should be raw user input, got: %s", content)
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
