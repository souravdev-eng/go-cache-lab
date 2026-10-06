package httpapi

import "github.com/gin-gonic/gin"

// GET /api/labs/hot-keys/replicated/books/:id
func (a *api) replicatedGet(c *gin.Context) {
	// TODO(replicated): select among logical Redis copies of the popular book.
	a.readFromPostgres(c)
}
