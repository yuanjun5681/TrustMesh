package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"trustmesh/backend/internal/store"
	"trustmesh/backend/internal/transport"
)

type DashboardHandler struct {
	store *store.Store
}

func NewDashboardHandler(s *store.Store) *DashboardHandler {
	return &DashboardHandler{store: s}
}

func (h *DashboardHandler) Stats(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	stats := h.store.GetDashboardStats(userID)
	transport.WriteData(c, http.StatusOK, stats)
}

func (h *DashboardHandler) RecentEvents(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	limit := queryInt(c, "limit", 20)
	events := h.store.ListUserEvents(userID, limit)
	transport.WriteList(c, events, len(events))
}

func (h *DashboardHandler) RecentTasks(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	limit := queryInt(c, "limit", 10)
	tasks := h.store.ListRecentTasks(userID, limit)
	transport.WriteList(c, tasks, len(tasks))
}

func (h *DashboardHandler) AgentEvents(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	limit := queryInt(c, "limit", 50)
	events, appErr := h.store.ListAgentEvents(userID, c.Param("id"), limit)
	if appErr != nil {
		transport.WriteError(c, appErr)
		return
	}
	transport.WriteList(c, events, len(events))
}

func queryInt(c *gin.Context, key string, defaultVal int) int {
	raw := c.Query(key)
	if raw == "" {
		return defaultVal
	}
	v, err := strconv.Atoi(raw)
	if err != nil || v <= 0 {
		return defaultVal
	}
	return v
}
