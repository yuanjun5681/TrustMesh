package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"trustmesh/backend/internal/clawsynapse"
	"trustmesh/backend/internal/transport"
)

type ClawSynapseHandler struct {
	client *clawsynapse.Client
}

func NewClawSynapseHandler(client *clawsynapse.Client) *ClawSynapseHandler {
	return &ClawSynapseHandler{client: client}
}

func (h *ClawSynapseHandler) Health(c *gin.Context) {
	health, err := h.client.GetHealth(c.Request.Context())
	if err != nil {
		transport.WriteData(c, http.StatusOK, gin.H{
			"online": false,
			"error":  err.Error(),
		})
		return
	}
	transport.WriteData(c, http.StatusOK, gin.H{
		"online":     true,
		"node_id":    health.Self.NodeID,
		"did":        health.Self.DID,
		"trust_mode": health.Self.TrustMode,
	})
}
