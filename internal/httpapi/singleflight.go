package httpapi

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Singleflight problem: simultaneous cold reads of one book each hit PostgreSQL.
// Starter behavior: every request bypasses Redis and causes a database load.
// Exercise: let same-book requests share one load, while other IDs stay independent.
// GET /api/labs/singleflight/books/:id
func (a *api) singleflightGet(c *gin.Context) {
	// TODO(singleflight): use a.cache.Get/Set with per-book coalescing.
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
