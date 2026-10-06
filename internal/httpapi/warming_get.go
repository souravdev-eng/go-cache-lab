package httpapi

import "github.com/gin-gonic/gin"

// GET /api/labs/warming/books/:id
func (a *api) warmingGet(c *gin.Context) {
	// TODO(warming): read the warming namespace; use PostgreSQL on a cold read.
	a.readFromPostgres(c)
}
