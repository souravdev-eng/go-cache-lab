package httpapi

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// GET /health
func (a *api) health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
