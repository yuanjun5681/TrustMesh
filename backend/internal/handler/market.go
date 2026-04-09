package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"trustmesh/backend/internal/model"
	"trustmesh/backend/internal/store"
	"trustmesh/backend/internal/transport"
)

type MarketHandler struct {
	market *store.MarketStore
}

func NewMarketHandler(ms *store.MarketStore) *MarketHandler {
	return &MarketHandler{market: ms}
}

// ListDepts GET /api/v1/market/departments
func (h *MarketHandler) ListDepts(c *gin.Context) {
	items := h.market.ListDepts()
	transport.WriteList(c, items, len(items))
}

// ListRoles GET /api/v1/market/roles?dept=engineering&q=后端
func (h *MarketHandler) ListRoles(c *gin.Context) {
	filter := model.MarketRoleFilter{
		DeptID: strings.TrimSpace(c.Query("dept")),
		Query:  strings.TrimSpace(c.Query("q")),
	}
	items := h.market.ListRoles(filter)
	transport.WriteList(c, items, len(items))
}

// GetRole GET /api/v1/market/roles/:id
func (h *MarketHandler) GetRole(c *gin.Context) {
	role, appErr := h.market.GetRole(c.Param("id"))
	if appErr != nil {
		transport.WriteError(c, appErr)
		return
	}
	transport.WriteData(c, http.StatusOK, role)
}

// DownloadRole GET /api/v1/market/roles/:id/download
func (h *MarketHandler) DownloadRole(c *gin.Context) {
	if appErr := h.market.WriteRoleZip(c.Writer, c.Param("id")); appErr != nil {
		transport.WriteError(c, appErr)
	}
}
