package httpapi

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Warming problem: the first read after startup must load PostgreSQL.
// Starter behavior: every read bypasses Redis; the warm action is incomplete.
// Exercise: fill Redis before reads, then compare warm and cold first requests.
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

// POST /api/labs/warming repeats the warm action without restarting the API.
func (a *api) warmingPost(c *gin.Context) {
	// TODO(warming): load selected books into the warming Redis namespace.
	c.JSON(http.StatusNotImplemented, gin.H{"status": "incomplete", "exercise": "warming", "message": "TODO: load selected books into the warming Redis namespace"})
}
