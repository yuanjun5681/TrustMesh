package handler

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"trustmesh/backend/internal/store"
	"trustmesh/backend/internal/transport"
)

// ConnectionNotifier sends platform-specific connection lifecycle notifications
// via the ClawSynapse network. Satisfied by *clawsynapse.WebhookHandler.
type ConnectionNotifier interface {
	NotifyConnectionEstablished(ctx context.Context, platform, platformNodeID, remoteUserID string)
	NotifyConnectionRemoved(ctx context.Context, platform, platformNodeID, remoteUserID string)
}

type PlatformHandler struct {
	store    *store.Store
	notifier ConnectionNotifier
}

func NewPlatformHandler(s *store.Store, notifier ConnectionNotifier) *PlatformHandler {
	return &PlatformHandler{store: s, notifier: notifier}
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

	go h.notifier.NotifyConnectionEstablished(context.Background(), conn.Platform, conn.PlatformNodeID, conn.RemoteUserID)

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

	// Fetch remoteUserID before deleting so we can notify with it.
	conns := h.store.ListPlatformConnections(userID)
	var remoteUserID string
	for _, c := range conns {
		if c.Platform == platform && c.PlatformNodeID == platformNodeID {
			remoteUserID = c.RemoteUserID
			break
		}
	}

	if appErr := h.store.DeletePlatformConnection(userID, platform, platformNodeID); appErr != nil {
		transport.WriteError(c, appErr)
		return
	}

	if remoteUserID != "" {
		go h.notifier.NotifyConnectionRemoved(context.Background(), platform, platformNodeID, remoteUserID)
	}

	c.Status(http.StatusNoContent)
}
