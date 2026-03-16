package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"trustmesh/backend/internal/store"
	"trustmesh/backend/internal/transport"
)

type ProjectHandler struct {
	store *store.Store
}

func NewProjectHandler(s *store.Store) *ProjectHandler {
	return &ProjectHandler{store: s}
}

type createProjectRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	PMAgentID   string `json:"pm_agent_id"`
}

func (h *ProjectHandler) Create(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	var req createProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		transport.WriteError(c, transport.BadRequest("BAD_REQUEST", "invalid json body"))
		return
	}
	project, appErr := h.store.CreateProject(userID, req.Name, req.Description, req.PMAgentID)
	if appErr != nil {
		transport.WriteError(c, appErr)
		return
	}
	transport.WriteData(c, http.StatusCreated, project)
}

func (h *ProjectHandler) List(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	items := h.store.ListProjects(userID)
	transport.WriteList(c, items, len(items))
}

func (h *ProjectHandler) Get(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	project, appErr := h.store.GetProject(userID, c.Param("projectId"))
	if appErr != nil {
		transport.WriteError(c, appErr)
		return
	}
	transport.WriteData(c, http.StatusOK, project)
}

type updateProjectRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	PMAgentID   *string `json:"pm_agent_id"`
}

func (h *ProjectHandler) Update(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	var req updateProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		transport.WriteError(c, transport.BadRequest("BAD_REQUEST", "invalid json body"))
		return
	}
	if req.PMAgentID != nil {
		transport.WriteError(c, transport.Validation("pm_agent_id is immutable in this endpoint", map[string]any{"pm_agent_id": "not allowed to update"}))
		return
	}

	project, appErr := h.store.UpdateProject(userID, c.Param("projectId"), store.UpdateProjectInput{
		Name:        req.Name,
		Description: req.Description,
	})
	if appErr != nil {
		transport.WriteError(c, appErr)
		return
	}
	transport.WriteData(c, http.StatusOK, project)
}

func (h *ProjectHandler) Archive(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	project, appErr := h.store.ArchiveProject(userID, c.Param("projectId"))
	if appErr != nil {
		transport.WriteError(c, appErr)
		return
	}
	transport.WriteData(c, http.StatusOK, project)
}
