package httpapi

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// GET /health
func (a *api) health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// GET /ready checks dependencies; it is separate from cache exercise behavior.
func (a *api) ready(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()
	postgresErr := a.store.Ping(ctx)
	redisErr := a.cache.Ping(ctx)
	status := http.StatusOK
	dependencies := gin.H{"postgres": "ok", "redis": "ok"}
	if postgresErr != nil {
		dependencies["postgres"] = "unavailable"
		status = http.StatusServiceUnavailable
	}
	if redisErr != nil {
		dependencies["redis"] = "unavailable"
		status = http.StatusServiceUnavailable
	}
	c.JSON(status, gin.H{"dependencies": dependencies})
}
