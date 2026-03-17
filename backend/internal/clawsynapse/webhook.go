package clawsynapse

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"trustmesh/backend/internal/model"
	"trustmesh/backend/internal/store"
	"trustmesh/backend/internal/transport"
)

type WebhookHandler struct {
	store       *store.Store
	client      *Client
	localNodeID string
	log         *zap.Logger
}

type WebhookPayload struct {
	NodeID     string         `json:"nodeId"`
	Type       string         `json:"type"`
	From       string         `json:"from"`
	SessionKey string         `json:"sessionKey"`
	Message    string         `json:"message"`
	Metadata   map[string]any `json:"metadata"`
}

type conversationReplyPayload struct {
	ConversationID string `json:"conversation_id"`
	Content        string `json:"content"`
}

type taskCreatePayload struct {
	ProjectID      string                  `json:"project_id"`
	ConversationID string                  `json:"conversation_id"`
	Title          string                  `json:"title"`
	Description    string                  `json:"description"`
	Todos          []taskCreateTodoPayload `json:"todos"`
}

type taskCreateTodoPayload struct {
	ID             string `json:"id"`
	Title          string `json:"title"`
	Description    string `json:"description"`
	AssigneeNodeID string `json:"assignee_node_id"`
}

type todoProgressPayload struct {
	TaskID  string `json:"task_id"`
	TodoID  string `json:"todo_id"`
	Message string `json:"message"`
}

type todoCompletePayload struct {
	TaskID string           `json:"task_id"`
	TodoID string           `json:"todo_id"`
	Result model.TodoResult `json:"result"`
}

type todoFailPayload struct {
	TaskID string `json:"task_id"`
	TodoID string `json:"todo_id"`
	Error  string `json:"error"`
}

type taskCreatedPayload struct {
	TaskID         string `json:"task_id"`
	ProjectID      string `json:"project_id"`
	ConversationID string `json:"conversation_id"`
	Title          string `json:"title"`
}

type taskUpdatedPayload struct {
	TaskID string `json:"task_id"`
	Status string `json:"status"`
}

type todoAssignedPayload struct {
	TaskID      string `json:"task_id"`
	TodoID      string `json:"todo_id"`
	Title       string `json:"title"`
	Description string `json:"description"`
}

type todoUpdatedPayload struct {
	TaskID  string `json:"task_id"`
	TodoID  string `json:"todo_id"`
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

func NewWebhookHandler(st *store.Store, client *Client, localNodeID string, log *zap.Logger) *WebhookHandler {
	return &WebhookHandler{
		store:       st,
		client:      client,
		localNodeID: strings.TrimSpace(localNodeID),
		log:         log,
	}
}

func (h *WebhookHandler) HandleWebhook(c *gin.Context) {
	var payload WebhookPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		transport.WriteError(c, transport.BadRequest("BAD_PAYLOAD", "invalid webhook payload"))
		return
	}

	if h.localNodeID != "" && payload.NodeID != "" && payload.NodeID != h.localNodeID {
		transport.WriteError(c, transport.Validation("invalid webhook target node", map[string]any{"nodeId": "does not match local node"}))
		return
	}

	switch strings.TrimSpace(payload.Type) {
	case "conversation.reply":
		h.handleConversationReply(c, payload)
	case "task.create":
		h.handleTaskCreate(c, payload)
	case "todo.progress":
		h.handleTodoProgress(c, payload)
	case "todo.complete":
		h.handleTodoComplete(c, payload)
	case "todo.fail":
		h.handleTodoFail(c, payload)
	default:
		transport.WriteError(c, transport.BadRequest("BAD_PAYLOAD", "unsupported webhook type"))
	}
}

func (h *WebhookHandler) handleConversationReply(c *gin.Context, webhook WebhookPayload) {
	var payload conversationReplyPayload
	if err := decodeWebhookMessage(webhook.Message, &payload); err != nil {
		transport.WriteError(c, transport.BadRequest("BAD_PAYLOAD", "invalid conversation.reply message"))
		return
	}

	detail, appErr := h.store.AppendPMReplyByNode(webhook.From, payload.ConversationID, payload.Content)
	if appErr != nil {
		transport.WriteError(c, appErr)
		return
	}
	transport.WriteData(c, http.StatusOK, detail)
}

func (h *WebhookHandler) handleTaskCreate(c *gin.Context, webhook WebhookPayload) {
	var payload taskCreatePayload
	if err := decodeWebhookMessage(webhook.Message, &payload); err != nil {
		transport.WriteError(c, transport.BadRequest("BAD_PAYLOAD", "invalid task.create message"))
		return
	}

	in := store.TaskCreateInput{
		ProjectID:      payload.ProjectID,
		ConversationID: payload.ConversationID,
		Title:          payload.Title,
		Description:    payload.Description,
		Todos:          make([]store.TaskCreateTodoInput, 0, len(payload.Todos)),
	}
	for _, todo := range payload.Todos {
		in.Todos = append(in.Todos, store.TaskCreateTodoInput{
			ID:             todo.ID,
			Title:          todo.Title,
			Description:    todo.Description,
			AssigneeNodeID: todo.AssigneeNodeID,
		})
	}

	task, appErr := h.store.CreateTaskByPMNodeWithMessageID(webhook.From, messageIDFromMetadata(webhook.Metadata), in)
	if appErr != nil {
		transport.WriteError(c, appErr)
		return
	}

	h.publishTaskCreated(task)
	for _, todo := range task.Todos {
		h.publish(context.Background(), todo.Assignee.NodeID, "todo.assigned", todoAssignedPayload{
			TaskID:      task.ID,
			TodoID:      todo.ID,
			Title:       todo.Title,
			Description: todo.Description,
		}, task.ID)
	}
	transport.WriteData(c, http.StatusOK, task)
}

func (h *WebhookHandler) handleTodoProgress(c *gin.Context, webhook WebhookPayload) {
	var payload todoProgressPayload
	if err := decodeWebhookMessage(webhook.Message, &payload); err != nil {
		transport.WriteError(c, transport.BadRequest("BAD_PAYLOAD", "invalid todo.progress message"))
		return
	}

	task, appErr := h.store.UpdateTodoProgressByNode(webhook.From, store.TodoProgressInput{
		TaskID:  payload.TaskID,
		TodoID:  payload.TodoID,
		Message: payload.Message,
	})
	if appErr != nil {
		transport.WriteError(c, appErr)
		return
	}

	h.publishTaskAndTodoUpdates(task, payload.TodoID, payload.Message)
	transport.WriteData(c, http.StatusOK, task)
}

func (h *WebhookHandler) handleTodoComplete(c *gin.Context, webhook WebhookPayload) {
	var payload todoCompletePayload
	if err := decodeWebhookMessage(webhook.Message, &payload); err != nil {
		transport.WriteError(c, transport.BadRequest("BAD_PAYLOAD", "invalid todo.complete message"))
		return
	}

	task, appErr := h.store.CompleteTodoByNodeWithMessageID(webhook.From, messageIDFromMetadata(webhook.Metadata), store.TodoCompleteInput{
		TaskID: payload.TaskID,
		TodoID: payload.TodoID,
		Result: payload.Result,
	})
	if appErr != nil {
		transport.WriteError(c, appErr)
		return
	}

	h.publishTaskAndTodoUpdates(task, payload.TodoID, "completed")
	transport.WriteData(c, http.StatusOK, task)
}

func (h *WebhookHandler) handleTodoFail(c *gin.Context, webhook WebhookPayload) {
	var payload todoFailPayload
	if err := decodeWebhookMessage(webhook.Message, &payload); err != nil {
		transport.WriteError(c, transport.BadRequest("BAD_PAYLOAD", "invalid todo.fail message"))
		return
	}

	task, appErr := h.store.FailTodoByNodeWithMessageID(webhook.From, messageIDFromMetadata(webhook.Metadata), store.TodoFailInput{
		TaskID: payload.TaskID,
		TodoID: payload.TodoID,
		Error:  payload.Error,
	})
	if appErr != nil {
		transport.WriteError(c, appErr)
		return
	}

	h.publishTaskAndTodoUpdates(task, payload.TodoID, payload.Error)
	transport.WriteData(c, http.StatusOK, task)
}

func (h *WebhookHandler) publishTaskCreated(task *model.TaskDetail) {
	h.publish(context.Background(), task.PMAgent.NodeID, "task.created", taskCreatedPayload{
		TaskID:         task.ID,
		ProjectID:      task.ProjectID,
		ConversationID: task.ConversationID,
		Title:          task.Title,
	}, task.ID)
}

func (h *WebhookHandler) publishTaskAndTodoUpdates(task *model.TaskDetail, todoID, message string) {
	h.publish(context.Background(), task.PMAgent.NodeID, "task.updated", taskUpdatedPayload{
		TaskID: task.ID,
		Status: task.Status,
	}, task.ID)

	todo := findTodo(task, todoID)
	if todo == nil {
		return
	}

	payload := todoUpdatedPayload{
		TaskID:  task.ID,
		TodoID:  todo.ID,
		Status:  todo.Status,
		Message: message,
	}
	h.publish(context.Background(), todo.Assignee.NodeID, "todo.updated", payload, task.ID)
	if task.PMAgent.NodeID != todo.Assignee.NodeID {
		h.publish(context.Background(), task.PMAgent.NodeID, "todo.updated", payload, task.ID)
	}
}

func (h *WebhookHandler) publish(ctx context.Context, targetNode, msgType string, payload any, sessionKey string) {
	if h.client == nil {
		return
	}
	if _, err := h.client.Publish(ctx, targetNode, msgType, payload, sessionKey, nil); err != nil && h.log != nil {
		h.log.Warn("clawsynapse publish failed", zap.String("target_node", targetNode), zap.String("type", msgType), zap.Error(err))
	}
}

func decodeWebhookMessage(raw string, out any) error {
	return json.Unmarshal([]byte(raw), out)
}

func messageIDFromMetadata(metadata map[string]any) string {
	if metadata == nil {
		return ""
	}
	for _, key := range []string{"messageId", "message_id", "id"} {
		if value, ok := metadata[key].(string); ok {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func findTodo(task *model.TaskDetail, todoID string) *model.Todo {
	for i := range task.Todos {
		if task.Todos[i].ID == todoID {
			return &task.Todos[i]
		}
	}
	return nil
}
