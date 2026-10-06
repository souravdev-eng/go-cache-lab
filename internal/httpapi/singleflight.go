package httpapi

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/sauravmajumdar/go-cache-lab/internal/bookstore"
)

// Singleflight problem: simultaneous cold reads of one book each hit PostgreSQL.
// Starter behavior: every request bypasses Redis and causes a database load.
// Exercise: let same-book requests share one load, while other IDs stay independent.
// GET /api/labs/singleflight/books/:id
func (a *api) singleflightGet(c *gin.Context) {
	id, ok := bookID(c)
	if !ok {
		return
	}

	key := "singleflight:book:" + strconv.FormatInt(id, 10)
	cached, err := a.cache.Get(c.Request.Context(), key)
	if err != nil {
		slog.Error("cache get failed", "error", err)
	} else {
		var book bookstore.Book
		if err := json.Unmarshal([]byte(cached), &book); err != nil {
			slog.Error("cache decode failed", "error", err)
		} else {
			c.Header("X-Cache-Result", "hit")
			slog.Info("book read", "book_id", id, "source", "redis", "cache_result", "hit")
			c.JSON(http.StatusOK, book)
			return
		}
	}

	bookObj, err := a.store.Get(c.Request.Context(), id)
	if err != nil {
		storeError(c, err)
		return
	}
	c.Header("X-Cache-Result", "bypass")
	encoded, err := json.Marshal(bookObj)
	if err != nil {
		slog.Error("cache encode failed", "error", err)
	} else if err := a.cache.Set(c.Request.Context(), key, string(encoded), 0); err != nil {
		slog.Error("cache set failed", "error", err)
	}
	slog.Info("book read", "book_id", id, "source", "postgres", "cache_result", "bypass")
	c.JSON(http.StatusOK, bookObj)
}
