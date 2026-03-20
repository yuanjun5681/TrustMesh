package clawsynapse

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"trustmesh/backend/internal/model"
	"trustmesh/backend/internal/protocol"
	"trustmesh/backend/internal/store"
	"trustmesh/backend/internal/transport"
)

type WebhookHandler struct {
	store       *store.Store
	client      *Client
	localNodeID string
	log         *zap.Logger
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
	var payload protocol.WebhookPayload
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

func (h *WebhookHandler) handleConversationReply(c *gin.Context, webhook protocol.WebhookPayload) {
	var payload protocol.ConversationReplyPayload
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

func (h *WebhookHandler) handleTaskCreate(c *gin.Context, webhook protocol.WebhookPayload) {
	var payload protocol.TaskCreatePayload
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
		h.publish(context.Background(), todo.Assignee.NodeID, "todo.assigned", protocol.TodoAssignedPayload{
			TaskID:      task.ID,
			TodoID:      todo.ID,
			Title:       todo.Title,
			Description: todo.Description,
			Content:     "你收到了一个新的 Todo 任务。请使用 /tm-task-exec skill 执行此任务，按要求回报进度和结果。",
			ExecBrief:   defaultExecBrief(),
		}, task.ID)
	}
	transport.WriteData(c, http.StatusOK, task)
}

func (h *WebhookHandler) handleTodoProgress(c *gin.Context, webhook protocol.WebhookPayload) {
	var payload protocol.TodoProgressPayload
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

	h.publishTaskAndTodoStatusChanges(task, payload.TodoID, payload.Message, webhook.From, "todo.progress")
	transport.WriteData(c, http.StatusOK, task)
}

func (h *WebhookHandler) handleTodoComplete(c *gin.Context, webhook protocol.WebhookPayload) {
	var payload protocol.TodoCompletePayload
	if err := decodeWebhookMessage(webhook.Message, &payload); err != nil {
		transport.WriteError(c, transport.BadRequest("BAD_PAYLOAD", "invalid todo.complete message"))
		return
	}
	h.enrichTodoResultTransfers(c.Request.Context(), &payload.Result)

	task, appErr := h.store.CompleteTodoByNodeWithMessageID(webhook.From, messageIDFromMetadata(webhook.Metadata), store.TodoCompleteInput{
		TaskID: payload.TaskID,
		TodoID: payload.TodoID,
		Result: payload.Result,
	})
	if appErr != nil {
		transport.WriteError(c, appErr)
		return
	}

	h.publishTaskAndTodoStatusChanges(task, payload.TodoID, "completed", webhook.From, "todo.complete")
	transport.WriteData(c, http.StatusOK, task)
}

func (h *WebhookHandler) handleTodoFail(c *gin.Context, webhook protocol.WebhookPayload) {
	var payload protocol.TodoFailPayload
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

	h.publishTaskAndTodoStatusChanges(task, payload.TodoID, payload.Error, webhook.From, "todo.fail")
	transport.WriteData(c, http.StatusOK, task)
}

func (h *WebhookHandler) enrichTodoResultTransfers(ctx context.Context, result *model.TodoResult) {
	if h == nil || h.client == nil || result == nil {
		return
	}

	transfersByID, orderedIDs := normalizeTransfers(result.Metadata)
	for _, ref := range result.ArtifactRefs {
		if strings.TrimSpace(ref.Kind) != "file" {
			continue
		}
		transferID := strings.TrimSpace(ref.ArtifactID)
		if transferID == "" {
			continue
		}
		if _, ok := transfersByID[transferID]; !ok {
			transfersByID[transferID] = map[string]any{"transfer_id": transferID}
			orderedIDs = append(orderedIDs, transferID)
		}
	}

	if len(transfersByID) == 0 {
		return
	}

	for _, transferID := range orderedIDs {
		transfer, ok := transfersByID[transferID]
		if !ok {
			continue
		}
		detail, err := h.client.GetTransfer(ctx, transferID)
		if err != nil {
			if h.log != nil {
				h.log.Warn("failed to enrich transfer details", zap.String("transfer_id", transferID), zap.Error(err))
			}
			continue
		}
		for key, value := range detail {
			transfer[key] = value
		}
		if _, ok := transfer["transfer_id"]; !ok {
			transfer["transfer_id"] = transferID
		}
	}

	if result.Metadata == nil {
		result.Metadata = map[string]any{}
	}
	items := make([]any, 0, len(orderedIDs))
	for _, transferID := range orderedIDs {
		if transfer, ok := transfersByID[transferID]; ok {
			items = append(items, transfer)
		}
	}
	result.Metadata["transfers"] = items
}

func normalizeTransfers(metadata map[string]any) (map[string]map[string]any, []string) {
	out := make(map[string]map[string]any)
	orderedIDs := make([]string, 0)
	if len(metadata) == 0 {
		return out, orderedIDs
	}

	rawTransfers, ok := metadata["transfers"]
	if !ok {
		return out, orderedIDs
	}
	items, ok := rawTransfers.([]any)
	if !ok {
		return out, orderedIDs
	}
	for _, item := range items {
		transfer, ok := item.(map[string]any)
		if !ok {
			continue
		}
		transferID := strings.TrimSpace(transferIDFromMap(transfer))
		if transferID == "" {
			continue
		}
		copyTransfer := make(map[string]any, len(transfer))
		for key, value := range transfer {
			copyTransfer[key] = value
		}
		out[transferID] = copyTransfer
		orderedIDs = append(orderedIDs, transferID)
	}
	return out, orderedIDs
}

func transferIDFromMap(transfer map[string]any) string {
	if transfer == nil {
		return ""
	}
	if v, ok := transfer["transfer_id"].(string); ok && strings.TrimSpace(v) != "" {
		return strings.TrimSpace(v)
	}
	if v, ok := transfer["transferId"].(string); ok && strings.TrimSpace(v) != "" {
		return strings.TrimSpace(v)
	}
	return ""
}

func (h *WebhookHandler) publishTaskCreated(task *model.TaskDetail) {
	h.publish(context.Background(), task.PMAgent.NodeID, "task.created", protocol.TaskCreatedPayload{
		TaskID:         task.ID,
		ProjectID:      task.ProjectID,
		ConversationID: task.ConversationID,
		Title:          task.Title,
	}, task.ID)
}

func (h *WebhookHandler) publishTaskAndTodoStatusChanges(task *model.TaskDetail, todoID, message, actorNodeID, cause string) {
	h.publish(context.Background(), task.PMAgent.NodeID, "task.status_changed", protocol.TaskStatusChangedPayload{
		TaskID:      task.ID,
		Status:      task.Status,
		ActorNodeID: strings.TrimSpace(actorNodeID),
		Cause:       strings.TrimSpace(cause),
		Version:     task.Version,
	}, task.ID)

	todo := findTodo(task, todoID)
	if todo == nil {
		return
	}

	payload := protocol.TodoStatusChangedPayload{
		TaskID:      task.ID,
		TodoID:      todo.ID,
		Status:      todo.Status,
		ActorNodeID: strings.TrimSpace(actorNodeID),
		Cause:       strings.TrimSpace(cause),
		Version:     task.Version,
		Message:     message,
	}
	if task.PMAgent.NodeID != "" {
		h.publish(context.Background(), task.PMAgent.NodeID, "todo.status_changed", payload, task.ID)
	}
	if todo.Assignee.NodeID != "" && todo.Assignee.NodeID != strings.TrimSpace(actorNodeID) && todo.Assignee.NodeID != task.PMAgent.NodeID {
		h.publish(context.Background(), todo.Assignee.NodeID, "todo.status_changed", payload, task.ID)
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

func defaultExecBrief() *protocol.TodoExecBrief {
	return &protocol.TodoExecBrief{
		Objective:    "执行分派的 Todo 任务；及时回报进度；完成后提交结果，失败时说明原因。",
		MustUseSkill: "tm-task-exec",
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
