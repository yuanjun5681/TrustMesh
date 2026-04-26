package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"trustmesh/backend/internal/store"
	"trustmesh/backend/internal/transport"
)


type PlatformHandler struct {
	store *store.Store
}

func NewPlatformHandler(s *store.Store) *PlatformHandler {
	return &PlatformHandler{store: s}
}

// Upsert godoc
// POST /api/v1/platform-connections
func (h *PlatformHandler) Upsert(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	var body struct {
		Platform       string `json:"platform"         binding:"required"`
		PlatformNodeID string `json:"platform_node_id" binding:"required"`
		RemoteUserID   string `json:"remote_user_id"   binding:"required"`
		PMAgentID      string `json:"pm_agent_id"      binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		transport.WriteError(c, transport.BadRequest("BAD_PAYLOAD", "invalid platform connection payload"))
		return
	}

	conn, appErr := h.store.UpsertPlatformConnection(userID, store.UpsertPlatformConnectionInput{
		Platform:       body.Platform,
		PlatformNodeID: body.PlatformNodeID,
		RemoteUserID:   body.RemoteUserID,
		PMAgentID:      body.PMAgentID,
	})
	if appErr != nil {
		transport.WriteError(c, appErr)
		return
	}
	transport.WriteData(c, http.StatusOK, conn)
}

// List godoc
// GET /api/v1/platform-connections
func (h *PlatformHandler) List(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	conns := h.store.ListPlatformConnections(userID)
	transport.WriteList(c, conns, len(conns))
}

// Delete godoc
// DELETE /api/v1/platform-connections/:platform/:platformNodeId
func (h *PlatformHandler) Delete(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	platform := c.Param("platform")
	platformNodeID := c.Param("platformNodeId")

	if appErr := h.store.DeletePlatformConnection(userID, platform, platformNodeID); appErr != nil {
		transport.WriteError(c, appErr)
		return
	}
	c.Status(http.StatusNoContent)
}
