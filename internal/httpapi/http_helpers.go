package httpapi

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/sauravmajumdar/go-cache-lab/internal/bookstore"
)

// readFromPostgres is the starter behavior shared by the GET routes.
// Replace the call in one lab file at a time as you complete its exercise.
func (a *api) readFromPostgres(c *gin.Context) {
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

func bookID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "book id must be a positive integer"})
		return 0, false
	}
	return id, true
}

func storeError(c *gin.Context, err error) {
	if errors.Is(err, bookstore.ErrNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "book not found"})
		return
	}
	slog.Error("book store unavailable", "error", err)
	c.JSON(http.StatusServiceUnavailable, gin.H{"error": "book store unavailable"})
}
