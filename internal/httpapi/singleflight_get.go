package httpapi

import "github.com/gin-gonic/gin"

// GET /api/labs/singleflight/books/:id
func (a *api) singleflightGet(c *gin.Context) {
	// TODO(singleflight): use a.cache.Get/Set with per-book coalescing.
	a.readFromPostgres(c)
}
