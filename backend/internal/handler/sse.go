package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

const sseHeartbeatInterval = 15 * time.Second

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

	// Force-write HTTP headers immediately so the client's fetch() resolves.
	// Without this, Gin defers header writing until the first body write,
	// causing streams with no initial data (e.g. user event stream) to hang.
	c.Writer.WriteHeaderNow()
	flusher.Flush()

	heartbeat := time.NewTicker(sseHeartbeatInterval)
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
