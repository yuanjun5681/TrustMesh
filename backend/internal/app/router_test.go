package app

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"trustmesh/backend/internal/config"
	"trustmesh/backend/internal/logger"
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
