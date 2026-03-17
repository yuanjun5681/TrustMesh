package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"trustmesh/backend/internal/store"
	"trustmesh/backend/internal/transport"
)

type TaskHandler struct {
	store *store.Store
}

func NewTaskHandler(s *store.Store) *TaskHandler {
	return &TaskHandler{store: s}
}

func (h *TaskHandler) ListByProject(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	status := c.Query("status")
	items, appErr := h.store.ListTasks(userID, c.Param("projectId"), status)
	if appErr != nil {
		transport.WriteError(c, appErr)
		return
	}
	transport.WriteList(c, items, len(items))
}

func (h *TaskHandler) Get(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	task, appErr := h.store.GetTask(userID, c.Param("id"))
	if appErr != nil {
		transport.WriteError(c, appErr)
		return
	}
	transport.WriteData(c, http.StatusOK, task)
}

func (h *TaskHandler) ListEvents(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	events, appErr := h.store.ListTaskEvents(userID, c.Param("id"))
	if appErr != nil {
		transport.WriteError(c, appErr)
		return
	}
	transport.WriteList(c, events, len(events))
}
