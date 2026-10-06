package httpapi

import "github.com/gin-gonic/gin"

// GET /api/books/:id is the PostgreSQL-only baseline.
func (a *api) getBook(c *gin.Context) {
	a.readFromPostgres(c)
}
