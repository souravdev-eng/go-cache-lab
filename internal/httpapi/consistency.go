package httpapi

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sauravmajumdar/go-cache-lab/internal/bookstore"
)

// Consistency problem: a cached book can become stale after an update.
// Starter behavior: reads bypass Redis and updates write only to PostgreSQL.
// Exercise: add cache-aside reads and invalidate copies after a database update.
// GET /api/labs/consistency/books/:id
func (a *api) consistencyGet(c *gin.Context) {
	// TODO(consistency): use a.cache.Get/Set with a short TTL.
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

// PUT /api/labs/consistency/books/:id
func (a *api) consistencyPut(c *gin.Context) {
	id, ok := bookID(c)
	if !ok {
		return
	}
	var payload struct {
		Title      string `json:"title"`
		Author     string `json:"author"`
		PriceCents *int64 `json:"price_cents"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil || strings.TrimSpace(payload.Title) == "" || strings.TrimSpace(payload.Author) == "" || payload.PriceCents == nil || *payload.PriceCents < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "title, author, and price_cents are required; price_cents must be nonnegative"})
		return
	}
	input := bookstore.BookUpdate{Title: payload.Title, Author: payload.Author, PriceCents: *payload.PriceCents}
	book, err := a.store.Update(c.Request.Context(), id, input)
	if err != nil {
		storeError(c, err)
		return
	}
	// TODO(consistency): invalidate every lab cache after the database commit.
	c.JSON(http.StatusOK, book)
}
