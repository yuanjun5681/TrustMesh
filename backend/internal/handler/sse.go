package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func beginSSE(c *gin.Context) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache, no-transform")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")
	c.Status(http.StatusOK)
}

func streamEvents[T any](c *gin.Context, updates <-chan T) {
	beginSSE(c)

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	heartbeat := time.NewTicker(25 * time.Second)
	defer heartbeat.Stop()

	for {
		select {
		case <-c.Request.Context().Done():
			return
		case update, ok := <-updates:
			if !ok {
				return
			}
			c.SSEvent("snapshot", update)
			flusher.Flush()
		case <-heartbeat.C:
			c.SSEvent("ping", gin.H{"ts": time.Now().UTC()})
			flusher.Flush()
		}
	}
}
