package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
	"trustmesh/backend/internal/auth"
	"trustmesh/backend/internal/transport"
)

const userIDKey = "user_id"

func UserID(c *gin.Context) string {
	if v, ok := c.Get(userIDKey); ok {
		if id, ok := v.(string); ok {
			return id
		}
	}
	return ""
}

func RequireAuth(jwtManager *auth.JWTManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		head := c.GetHeader("Authorization")
		if !strings.HasPrefix(strings.ToLower(head), "bearer ") {
			transport.WriteError(c, transport.Unauthorized("missing or invalid authorization header"))
			c.Abort()
			return
		}
		raw := strings.TrimSpace(head[7:])
		claims, err := jwtManager.ParseToken(raw)
		if err != nil || claims.UserID == "" {
			transport.WriteError(c, transport.Unauthorized("invalid or expired token"))
			c.Abort()
			return
		}
		c.Set(userIDKey, claims.UserID)
		c.Next()
	}
}
