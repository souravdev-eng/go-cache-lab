package httpapi

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
)

// GET /api/labs/warming/books/:id
func (a *api) warmingGet(c *gin.Context) {
	// TODO(warming): read the warming namespace; use PostgreSQL on a cold read.
	id, ok := bookID(c)
	if !ok {
		return
	}
	book, err := a.store.Get(c.Request.Context(), id)
	if err != nil {
		storeError(c, err)
		return
	}
	c.Header("X-Cache-Result", "bypass")
	slog.Info("book read", "book_id", id, "route", c.FullPath(), "source", "postgres", "cache_result", "bypass")
	c.JSON(http.StatusOK, book)
}
