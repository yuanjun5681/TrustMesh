package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"trustmesh/backend/internal/middleware"
	"trustmesh/backend/internal/transport"
)

func currentUserID(c *gin.Context) (string, bool) {
	uid := middleware.UserID(c)
	if uid == "" {
		transport.WriteError(c, &transport.AppError{Status: http.StatusUnauthorized, Code: "UNAUTHORIZED", Message: "missing auth context", Details: map[string]any{}})
		return "", false
	}
	return uid, true
}
