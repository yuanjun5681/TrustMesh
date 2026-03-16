package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"trustmesh/backend/internal/store"
	"trustmesh/backend/internal/transport"
)

type AgentHandler struct {
	store *store.Store
}

func NewAgentHandler(s *store.Store) *AgentHandler {
	return &AgentHandler{store: s}
}

type createAgentRequest struct {
	NodeID       string   `json:"node_id"`
	Name         string   `json:"name"`
	Role         string   `json:"role"`
	Description  string   `json:"description"`
	Capabilities []string `json:"capabilities"`
}

func (h *AgentHandler) Create(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	var req createAgentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		transport.WriteError(c, transport.BadRequest("BAD_REQUEST", "invalid json body"))
		return
	}

	agent, appErr := h.store.CreateAgent(userID, req.NodeID, req.Name, req.Role, req.Description, req.Capabilities)
	if appErr != nil {
		transport.WriteError(c, appErr)
		return
	}
	transport.WriteData(c, http.StatusCreated, agent)
}

func (h *AgentHandler) List(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	items := h.store.ListAgents(userID)
	transport.WriteList(c, items, len(items))
}

func (h *AgentHandler) Get(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	agent, appErr := h.store.GetAgent(userID, c.Param("id"))
	if appErr != nil {
		transport.WriteError(c, appErr)
		return
	}
	transport.WriteData(c, http.StatusOK, agent)
}

type updateAgentRequest struct {
	Name         *string   `json:"name"`
	Role         *string   `json:"role"`
	Description  *string   `json:"description"`
	Capabilities *[]string `json:"capabilities"`
	NodeID       *string   `json:"node_id"`
}

func (h *AgentHandler) Update(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	var req updateAgentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		transport.WriteError(c, transport.BadRequest("BAD_REQUEST", "invalid json body"))
		return
	}
	if req.NodeID != nil {
		transport.WriteError(c, transport.Validation("node_id is immutable", map[string]any{"node_id": "not allowed to update"}))
		return
	}

	agent, appErr := h.store.UpdateAgent(userID, c.Param("id"), store.UpdateAgentInput{
		Name:         req.Name,
		Role:         req.Role,
		Description:  req.Description,
		Capabilities: req.Capabilities,
	})
	if appErr != nil {
		transport.WriteError(c, appErr)
		return
	}
	transport.WriteData(c, http.StatusOK, agent)
}

func (h *AgentHandler) Delete(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	if appErr := h.store.DeleteAgent(userID, c.Param("id")); appErr != nil {
		transport.WriteError(c, appErr)
		return
	}
	c.Status(http.StatusNoContent)
}
