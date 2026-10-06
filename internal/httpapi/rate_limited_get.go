package httpapi

import "github.com/gin-gonic/gin"

// GET /api/labs/hot-keys/rate-limited/books/:id
func (a *api) rateLimitedGet(c *gin.Context) {
	// TODO(rate-limited): check a per-book allowance before loading the book.
	a.readFromPostgres(c)
}
