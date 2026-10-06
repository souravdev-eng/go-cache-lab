package httpapi

import "github.com/gin-gonic/gin"

// GET /api/labs/consistency/books/:id
func (a *api) consistencyGet(c *gin.Context) {
	// TODO(consistency): use a.cache.Get/Set with a short TTL.
	a.readFromPostgres(c)
}
