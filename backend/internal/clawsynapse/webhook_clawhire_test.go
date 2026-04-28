package clawsynapse

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"trustmesh/backend/internal/model"
	"trustmesh/backend/internal/store"
)

func TestNotifyClawHireSubmissionWaitsForDeclaredTransfers(t *testing.T) {
	var (
		mu       sync.Mutex
		requests []map[string]any
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/publish" {
			http.NotFound(w, r)
			return
		}
		defer r.Body.Close()
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode publish payload: %v", err)
		}
		mu.Lock()
		requests = append(requests, payload)
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"code":"msg.published","message":"ok","data":{"targetNode":"node-clawhire","messageId":"msg-1"},"ts":1}`))
	}))
	defer server.Close()

	handler := NewWebhookHandler(store.New(), NewClient(server.URL, time.Second), nil, "https://trustmesh.example")
	task := clawHireDoneTaskWithArtifactRef()

	handler.NotifyClawHireSubmission(context.Background(), task)
	if got := len(requests); got != 0 {
		t.Fatalf("expected no submission before transfer.received, got %d publish requests", got)
	}

	task.Artifacts = []model.TaskArtifact{{TransferID: "tf_login_guide", TaskID: task.ID, FileName: "login-guide.md"}}
	handler.NotifyClawHireSubmission(context.Background(), task)

	mu.Lock()
	defer mu.Unlock()
	if len(requests) != 1 {
		t.Fatalf("expected submission after transfer arrived, got %d publish requests", len(requests))
	}
	req := requests[0]
	if req["type"] != "clawhire.submission.created" || req["targetNode"] != "node-clawhire" {
		t.Fatalf("unexpected publish request: %#v", req)
	}
	var message map[string]any
	rawMessage, _ := req["message"].(string)
	if err := json.Unmarshal([]byte(rawMessage), &message); err != nil {
		t.Fatalf("unmarshal submission message: %v", err)
	}
	artifacts, _ := message["artifacts"].([]any)
	if len(artifacts) != 1 {
		t.Fatalf("expected one artifact in submission, got %#v", message["artifacts"])
	}
	first, _ := artifacts[0].(map[string]any)
	if first["url"] != "https://trustmesh.example/public/tasks/task-local/artifacts/tf_login_guide/content" {
		t.Fatalf("unexpected artifact url: %#v", first)
	}
}

func clawHireDoneTaskWithArtifactRef() *model.TaskDetail {
	return &model.TaskDetail{
		ID:     "task-local",
		UserID: "user-trustmesh",
		Status: "done",
		ExternalRef: &model.ExternalTaskRef{
			Platform:       "clawhire",
			ExternalTaskID: "task-clawhire",
			RemoteUserID:   "acct-clawhire",
			PlatformNodeID: "node-clawhire",
			ContractID:     "contract-1",
		},
		Result: model.TaskResult{
			Summary:     "done",
			FinalOutput: "implemented login",
			Metadata:    map[string]any{},
		},
		Todos: []model.Todo{
			{
				ID:     "todo-1",
				Status: "done",
				Result: model.TodoResult{
					Summary: "done",
					Output:  "implemented login",
					ArtifactRefs: []model.TodoResultArtifactRef{
						{ArtifactID: "tf_login_guide", Kind: "file", Label: "Login guide"},
					},
				},
			},
		},
	}
}
