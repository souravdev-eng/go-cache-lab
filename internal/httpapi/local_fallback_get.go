package httpapi

import "github.com/gin-gonic/gin"

// GET /api/labs/hot-keys/local-fallback/books/:id
func (a *api) localFallbackGet(c *gin.Context) {
	// TODO(local-fallback): maintain a bounded local copy for Redis outages.
	a.readFromPostgres(c)
}
