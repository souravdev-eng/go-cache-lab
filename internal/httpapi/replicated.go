package httpapi

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Replicated hot-key problem: many readers contend for one popular cache key.
// Starter behavior: every read bypasses Redis and loads PostgreSQL.
// Exercise: create logical Redis copies and show which copy served a read.
// GET /api/labs/hot-keys/replicated/books/:id
func (a *api) replicatedGet(c *gin.Context) {
	// TODO(replicated): select among logical Redis copies of the popular book.
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
