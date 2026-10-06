package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/sauravmajumdar/go-cache-lab/internal/bookstore"
)

type bookFlight struct {
	done   chan struct{}
	book   bookstore.Book
	result string
	err    error
}

func (a *api) cachedBook(ctx context.Context, key string) (bookstore.Book, bool) {
	cached, err := a.cache.Get(ctx, key)
	if err != nil {
		if !errors.Is(err, redis.Nil) {
			slog.Error("cache get failed", "key", key, "error", err)
		}
		return bookstore.Book{}, false
	}
	var book bookstore.Book
	if err := json.Unmarshal([]byte(cached), &book); err != nil {
		slog.Error("cache decode failed", "key", key, "error", err)
		return bookstore.Book{}, false
	}
	return book, true
}

// doBookLoad only holds the mutex while managing flights; different keys load independently.
func (a *api) doBookLoad(ctx context.Context, key string, id int64) (bookstore.Book, string, error) {
	a.flightMu.Lock()
	if flight := a.flights[key]; flight != nil {
		a.flightMu.Unlock()
		select {
		case <-flight.done:
			if flight.result == "miss" {
				return flight.book, "shared", flight.err
			}
			return flight.book, flight.result, flight.err
		case <-ctx.Done():
			return bookstore.Book{}, "", ctx.Err()
		}
	}
	flight := &bookFlight{done: make(chan struct{})}
	a.flights[key] = flight
	a.flightMu.Unlock()

	// A request may have missed just before an earlier flight filled Redis.
	book, ok := a.cachedBook(ctx, key)
	if ok {
		flight.book, flight.result = book, "hit"
	} else {
		flight.book, flight.err = a.store.Get(ctx, id)
		if flight.err == nil {
			flight.result = "miss"
			encoded, err := json.Marshal(flight.book)
			if err != nil {
				slog.Error("cache encode failed", "error", err)
			} else if err := a.cache.Set(ctx, key, string(encoded), a.cacheTTL); err != nil {
				slog.Error("cache set failed", "key", key, "error", err)
			}
		}
	}
	a.flightMu.Lock()
	delete(a.flights, key)
	close(flight.done)
	a.flightMu.Unlock()
	return flight.book, flight.result, flight.err
}

// GET /api/labs/singleflight/books/:id
func (a *api) singleflightGet(c *gin.Context) {
	id, ok := bookID(c)
	if !ok {
		return
	}

	key := "singleflight:book:" + strconv.FormatInt(id, 10)
	if book, ok := a.cachedBook(c.Request.Context(), key); ok {
		c.Header("X-Cache-Result", "hit")
		slog.Info("book read", "book_id", id, "route", c.FullPath(), "source", "redis", "cache_result", "hit")
		c.JSON(http.StatusOK, book)
		return
	}

	bookObj, result, err := a.doBookLoad(c.Request.Context(), key, id)
	if err != nil {
		storeError(c, err)
		return
	}
	c.Header("X-Cache-Result", result)
	source := "postgres"
	if result == "hit" {
		source = "redis"
	} else if result == "shared" {
		source = "coalesced"
	}
	slog.Info("book read", "book_id", id, "route", c.FullPath(), "source", source, "cache_result", result)
	c.JSON(http.StatusOK, bookObj)
}
