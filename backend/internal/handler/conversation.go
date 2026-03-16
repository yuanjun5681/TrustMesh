package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"trustmesh/backend/internal/nats"
	"trustmesh/backend/internal/store"
	"trustmesh/backend/internal/transport"
)

type ConversationHandler struct {
	store     *store.Store
	publisher *nats.Publisher
	log       *zap.Logger
}

func NewConversationHandler(s *store.Store, publisher *nats.Publisher, log *zap.Logger) *ConversationHandler {
	return &ConversationHandler{store: s, publisher: publisher, log: log}
}

type createConversationRequest struct {
	Content string `json:"content"`
}

func (h *ConversationHandler) Create(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	var req createConversationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		transport.WriteError(c, transport.BadRequest("BAD_REQUEST", "invalid json body"))
		return
	}
	detail, appErr := h.store.CreateConversation(userID, c.Param("projectId"), req.Content)
	if appErr != nil {
		transport.WriteError(c, appErr)
		return
	}
	h.notifyPM(userID, detail.ProjectID, detail.ID, req.Content)
	transport.WriteData(c, http.StatusCreated, detail)
}

func (h *ConversationHandler) ListByProject(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	items, appErr := h.store.ListConversations(userID, c.Param("projectId"))
	if appErr != nil {
		transport.WriteError(c, appErr)
		return
	}
	transport.WriteList(c, items, len(items))
}

func (h *ConversationHandler) Get(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	detail, appErr := h.store.GetConversation(userID, c.Param("id"))
	if appErr != nil {
		transport.WriteError(c, appErr)
		return
	}
	transport.WriteData(c, http.StatusOK, detail)
}

func (h *ConversationHandler) AppendMessage(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	var req createConversationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		transport.WriteError(c, transport.BadRequest("BAD_REQUEST", "invalid json body"))
		return
	}
	detail, appErr := h.store.AppendConversationMessage(userID, c.Param("id"), req.Content)
	if appErr != nil {
		transport.WriteError(c, appErr)
		return
	}
	h.notifyPM(userID, detail.ProjectID, detail.ID, req.Content)
	transport.WriteData(c, http.StatusOK, detail)
}

func (h *ConversationHandler) notifyPM(userID, projectID, conversationID, content string) {
	if h.publisher == nil || h.log == nil {
		return
	}
	pmNodeID, appErr := h.store.GetProjectPMNode(userID, projectID)
	if appErr != nil {
		h.log.Warn("skip notify conversation.message", zap.String("project_id", projectID), zap.String("code", appErr.Code), zap.String("message", appErr.Message))
		return
	}
	if err := h.publisher.NotifyConversationMessage(pmNodeID, nats.ConversationMessagePayload{
		ConversationID: conversationID,
		ProjectID:      projectID,
		Content:        content,
	}); err != nil {
		h.log.Warn("notify conversation.message failed", zap.String("project_id", projectID), zap.String("conversation_id", conversationID), zap.Error(err))
	}
}
