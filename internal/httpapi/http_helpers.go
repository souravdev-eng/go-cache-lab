package httpapi

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/sauravmajumdar/go-cache-lab/internal/bookstore"
)

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
