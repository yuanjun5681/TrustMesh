package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"trustmesh/backend/internal/transport"
)

func Recovery(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if rec := recover(); rec != nil {
				logger.Error("panic recovered", zap.Any("panic", rec), zap.String("path", c.Request.URL.Path))
				transport.WriteError(c, &transport.AppError{
					Status:  http.StatusInternalServerError,
					Code:    "INTERNAL_ERROR",
					Message: "internal server error",
					Details: map[string]any{},
				})
				c.Abort()
			}
		}()
		c.Next()
	}
}
