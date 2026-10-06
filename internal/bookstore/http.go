package bookstore

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

var ErrNotFound = errors.New("book not found")

type Book struct {
	ID         int64     `json:"id"`
	Title      string    `json:"title"`
	Author     string    `json:"author"`
	PriceCents int64     `json:"price_cents"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type BookUpdate struct {
	Title      string `json:"title"`
	Author     string `json:"author"`
	PriceCents int64  `json:"price_cents"`
}

type BookStore interface {
	Get(context.Context, int64) (Book, error)
	Update(context.Context, int64, BookUpdate) (Book, error)
	Ping(context.Context) error
}

type Pinger interface{ Ping(context.Context) error }

// NewRouter exposes the working database path and independent starter routes.
// Each lab route deliberately bypasses Redis until its exercise is completed.
func NewRouter(store BookStore, redis Pinger) http.Handler {
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())
	r.GET("/health", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) })
	r.GET("/ready", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()
		postgresErr := store.Ping(ctx)
		redisErr := redis.Ping(ctx)
		status := http.StatusOK
		dependencies := gin.H{"postgres": "ok", "redis": "ok"}
		if postgresErr != nil {
			dependencies["postgres"] = "unavailable"
			status = http.StatusServiceUnavailable
		}
		if redisErr != nil {
			dependencies["redis"] = "unavailable"
			status = http.StatusServiceUnavailable
		}
		c.JSON(status, gin.H{"dependencies": dependencies})
	})

	readBook := func(c *gin.Context) {
		id, ok := bookID(c)
		if !ok {
			return
		}
		book, err := store.Get(c.Request.Context(), id)
		if err != nil {
			storeError(c, err)
			return
		}
		c.Header("X-Cache-Result", "bypass")
		slog.Info("book read", "book_id", id, "route", c.FullPath(), "source", "postgres", "cache_result", "bypass")
		c.JSON(http.StatusOK, book)
	}

	r.GET("/api/books/:id", readBook)
	// TODO(singleflight): replace this handler with cache aside and per-book coalescing.
	r.GET("/api/labs/singleflight/books/:id", readBook)
	// TODO(warming): read the warming namespace; keep PostgreSQL as the cold path.
	r.GET("/api/labs/warming/books/:id", readBook)
	// TODO(consistency): cache aside here, then invalidate after a successful update.
	r.GET("/api/labs/consistency/books/:id", readBook)
	// TODO(replicated): select among logical copies of the popular book.
	r.GET("/api/labs/hot-keys/replicated/books/:id", readBook)
	// TODO(local-fallback): maintain a bounded local copy for Redis outages.
	r.GET("/api/labs/hot-keys/local-fallback/books/:id", readBook)
	// TODO(rate-limited): check a per-book allowance before loading the book.
	r.GET("/api/labs/hot-keys/rate-limited/books/:id", readBook)
	r.POST("/api/labs/warming", func(c *gin.Context) {
		c.JSON(http.StatusNotImplemented, gin.H{"status": "incomplete", "exercise": "warming", "message": "TODO: load selected books into the warming Redis namespace"})
	})
	r.PUT("/api/labs/consistency/books/:id", func(c *gin.Context) {
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
		input := BookUpdate{Title: payload.Title, Author: payload.Author, PriceCents: *payload.PriceCents}
		book, err := store.Update(c.Request.Context(), id, input)
		if err != nil {
			storeError(c, err)
			return
		}
		// TODO(consistency): invalidate every lab cache after the database commit.
		c.JSON(http.StatusOK, book)
	})
	return r
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
	if errors.Is(err, ErrNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "book not found"})
		return
	}
	slog.Error("book store unavailable", "error", err)
	c.JSON(http.StatusServiceUnavailable, gin.H{"error": "book store unavailable"})
}
