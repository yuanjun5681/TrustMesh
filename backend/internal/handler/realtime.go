package handler

import (
	"github.com/gin-gonic/gin"
	"trustmesh/backend/internal/store"
)

type RealtimeHandler struct {
	store *store.Store
}

func NewRealtimeHandler(s *store.Store) *RealtimeHandler {
	return &RealtimeHandler{store: s}
}

func (h *RealtimeHandler) Stream(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	updates, unsubscribe := h.store.SubscribeUser(userID)
	defer unsubscribe()

	streamEvents(c, updates)
}
