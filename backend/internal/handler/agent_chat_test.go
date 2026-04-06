package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"
	"unsafe"

	"github.com/gin-gonic/gin"
	"trustmesh/backend/internal/clawsynapse"
	"trustmesh/backend/internal/store"
)

func TestSendMessagePublishesChatMessageType(t *testing.T) {
	gin.SetMode(gin.TestMode)

	st := store.New()
	user, appErr := st.CreateUser("user@example.com", "User", "hash")
	if appErr != nil {
		t.Fatalf("create user: %v", appErr)
	}
	agent, appErr := st.CreateAgent(user.ID, "node-chat-001", "Chat Agent", "developer", "dev", []string{"conversation"})
	if appErr != nil {
		t.Fatalf("create agent: %v", appErr)
	}
	now := time.Now().UTC()
	st.SyncAgentPresence([]store.AgentPresence{{NodeID: agent.NodeID, LastSeenAt: now}}, now)

	var publishReq map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&publishReq); err != nil {
			t.Fatalf("decode publish request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"code":"OK","message":"ok","data":{"targetNode":"node-chat-001","messageId":"remote-1"},"ts":1}`))
	}))
	defer server.Close()

	h := NewAgentChatHandler(st, clawsynapse.NewClient(server.URL, 0), nil)

	body, _ := json.Marshal(map[string]any{"content": "hello"})
	req := httptest.NewRequest(http.MethodPost, "/agents/"+agent.ID+"/chat/messages", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Params = gin.Params{{Key: "id", Value: agent.ID}}
	c.Set("user_id", user.ID)

	h.SendMessage(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if publishReq["type"] != "chat.message" {
		t.Fatalf("expected publish type chat.message, got %#v", publishReq["type"])
	}
	if publishReq["sessionKey"] == "" {
		t.Fatalf("expected sessionKey to be set, got %#v", publishReq["sessionKey"])
	}
}

func TestSendMessageReturnsSuccessWhenRemoteDeliverySucceededButLocalStatusUpdateFails(t *testing.T) {
	gin.SetMode(gin.TestMode)

	st := store.New()
	user, appErr := st.CreateUser("user@example.com", "User", "hash")
	if appErr != nil {
		t.Fatalf("create user: %v", appErr)
	}
	agent, appErr := st.CreateAgent(user.ID, "node-chat-001", "Chat Agent", "developer", "dev", []string{"conversation"})
	if appErr != nil {
		t.Fatalf("create agent: %v", appErr)
	}
	now := time.Now().UTC()
	st.SyncAgentPresence([]store.AgentPresence{{NodeID: agent.NodeID, LastSeenAt: now}}, now)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clearAgentChatsForTest(st)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"code":"OK","message":"ok","data":{"targetNode":"node-chat-001","messageId":"remote-1"},"ts":1}`))
	}))
	defer server.Close()

	h := NewAgentChatHandler(st, clawsynapse.NewClient(server.URL, 0), nil)

	body, _ := json.Marshal(map[string]any{"content": "hello"})
	req := httptest.NewRequest(http.MethodPost, "/agents/"+agent.ID+"/chat/messages", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Params = gin.Params{{Key: "id", Value: agent.ID}}
	c.Set("user_id", user.ID)

	h.SendMessage(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if !bytes.Contains(w.Body.Bytes(), []byte(`"status":"sent"`)) {
		t.Fatalf("expected sent status in response, got %s", w.Body.String())
	}
	if !bytes.Contains(w.Body.Bytes(), []byte(`"remote_message_id":"remote-1"`)) {
		t.Fatalf("expected remote message id in response, got %s", w.Body.String())
	}
}

func clearAgentChatsForTest(st *store.Store) {
	value := reflect.ValueOf(st).Elem().FieldByName("agentChats")
	writable := reflect.NewAt(value.Type(), unsafe.Pointer(value.UnsafeAddr())).Elem()
	for _, key := range writable.MapKeys() {
		writable.SetMapIndex(key, reflect.Value{})
	}
}
