package handler

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"trustmesh/backend/internal/clawsynapse"
	"trustmesh/backend/internal/store"
	"trustmesh/backend/internal/transport"
)

type AgentHandler struct {
	store      *store.Store
	clawClient *clawsynapse.Client
}

func NewAgentHandler(s *store.Store, clawClient *clawsynapse.Client) *AgentHandler {
	return &AgentHandler{store: s, clawClient: clawClient}
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
	if nodeID := strings.TrimSpace(req.NodeID); nodeID != "" {
		if appErr := h.ensureNodeOnline(c.Request.Context(), nodeID); appErr != nil {
			transport.WriteError(c, appErr)
			return
		}
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

func (h *AgentHandler) Stats(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	stats, appErr := h.store.GetAgentStats(userID, c.Param("id"))
	if appErr != nil {
		transport.WriteError(c, appErr)
		return
	}
	transport.WriteData(c, http.StatusOK, stats)
}

func (h *AgentHandler) ensureNodeOnline(ctx context.Context, nodeID string) *transport.AppError {
	if h.clawClient == nil {
		err := transport.NewError(http.StatusServiceUnavailable, "CLAWSYNAPSE_UNAVAILABLE", "暂时无法校验节点在线状态")
		err.Details = map[string]any{"node_id": nodeID}
		return err
	}

	peers, err := h.clawClient.GetPeers(ctx)
	if err != nil {
		appErr := transport.NewError(http.StatusServiceUnavailable, "CLAWSYNAPSE_UNAVAILABLE", "暂时无法校验节点在线状态")
		appErr.Details = map[string]any{
			"node_id": nodeID,
			"cause":   err.Error(),
		}
		return appErr
	}

	for _, peer := range peers {
		if strings.TrimSpace(peer.NodeID) == nodeID {
			return nil
		}
	}

	return transport.Validation("node_id 必须对应一个在线中的 ClawSynapse 节点", map[string]any{
		"node_id": "offline_or_not_found",
	})
}
