package httpapi

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// POST /api/labs/warming
func (a *api) warmingPost(c *gin.Context) {
	// TODO(warming): load selected books into the warming Redis namespace.
	c.JSON(http.StatusNotImplemented, gin.H{"status": "incomplete", "exercise": "warming", "message": "TODO: load selected books into the warming Redis namespace"})
}
