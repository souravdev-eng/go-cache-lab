package httpapi

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sauravmajumdar/go-cache-lab/internal/bookstore"
)

// BookStore lists the database methods needed by handlers. PostgresStore has
// these methods, so it satisfies the interface without an explicit declaration.
type BookStore interface {
	Get(context.Context, int64) (bookstore.Book, error)
	Update(context.Context, int64, bookstore.BookUpdate) (bookstore.Book, error)
	Ping(context.Context) error
}

// Cache is the Redis access available to each lab handler as exercises are filled in.
type Cache interface {
	Ping(context.Context) error
	Get(context.Context, string) (string, error)
	Set(context.Context, string, string, time.Duration) error
	Delete(context.Context, ...string) error
}

// api gives each route handler access to the same database and Redis clients.
type api struct {
	store BookStore
	cache Cache
}

// NewRouter keeps URLs here; each cache pattern's handlers live together in one file.
func NewRouter(store BookStore, cache Cache) http.Handler {
	a := &api{store: store, cache: cache}
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())
	// Process checks and the database-only comparison route.
	r.GET("/health", a.health)
	r.GET("/ready", a.ready)
	r.GET("/api/books/:id", a.getBook)

	// Open one pattern file to see its starter flow and focused TODOs.
	r.GET("/api/labs/singleflight/books/:id", a.singleflightGet)
	r.GET("/api/labs/warming/books/:id", a.warmingGet)
	r.POST("/api/labs/warming", a.warmingPost)
	r.GET("/api/labs/consistency/books/:id", a.consistencyGet)
	r.PUT("/api/labs/consistency/books/:id", a.consistencyPut)
	r.GET("/api/labs/hot-keys/replicated/books/:id", a.replicatedGet)
	r.GET("/api/labs/hot-keys/local-fallback/books/:id", a.localFallbackGet)
	r.GET("/api/labs/hot-keys/rate-limited/books/:id", a.rateLimitedGet)
	return r
}
