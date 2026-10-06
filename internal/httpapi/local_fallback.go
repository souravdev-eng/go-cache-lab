package httpapi

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Local fallback problem: Redis can be unavailable during a hot read.
// Starter behavior: PostgreSQL serves the book even when Redis is down.
// Exercise: retain a short-lived local copy and use it during Redis failures.
// GET /api/labs/hot-keys/local-fallback/books/:id
func (a *api) localFallbackGet(c *gin.Context) {
	// TODO(local-fallback): maintain a bounded local copy for Redis outages.
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
