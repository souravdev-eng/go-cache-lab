package httpapi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/sauravmajumdar/go-cache-lab/internal/bookstore"
	"github.com/sauravmajumdar/go-cache-lab/internal/httpapi"
)

var sampleBook = bookstore.Book{
	ID: 1, Title: "The Go Programming Language", Author: "Alan A. A. Donovan and Brian W. Kernighan",
	PriceCents: 3999, UpdatedAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
}

type memoryStore struct {
	books     map[int64]bookstore.Book
	pingErr   error
	getErr    error
	updateErr error
}

func (s *memoryStore) Get(_ context.Context, id int64) (bookstore.Book, error) {
	if s.getErr != nil {
		return bookstore.Book{}, s.getErr
	}
	b, ok := s.books[id]
	if !ok {
		return bookstore.Book{}, bookstore.ErrNotFound
	}
	return b, nil
}

func (s *memoryStore) Update(_ context.Context, id int64, input bookstore.BookUpdate) (bookstore.Book, error) {
	if s.updateErr != nil {
		return bookstore.Book{}, s.updateErr
	}
	b, ok := s.books[id]
	if !ok {
		return bookstore.Book{}, bookstore.ErrNotFound
	}
	b.Title, b.Author, b.PriceCents = input.Title, input.Author, input.PriceCents
	b.UpdatedAt = time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC)
	s.books[id] = b
	return b, nil
}

func (s *memoryStore) Ping(context.Context) error { return s.pingErr }

type pingService struct{ err error }

func (p pingService) Ping(context.Context) error                               { return p.err }
func (p pingService) Get(context.Context, string) (string, error)              { return "", p.err }
func (p pingService) Set(context.Context, string, string, time.Duration) error { return p.err }
func (p pingService) Delete(context.Context, ...string) error                  { return p.err }

func newRouter(store *memoryStore, redisErr error) http.Handler {
	return httpapi.NewRouter(store, pingService{redisErr})
}

type memoryCache struct{ values map[string]string }

func (c *memoryCache) Ping(context.Context) error { return nil }
func (c *memoryCache) Get(_ context.Context, key string) (string, error) {
	value, ok := c.values[key]
	if !ok {
		return "", errors.New("cache miss")
	}
	return value, nil
}
func (c *memoryCache) Set(_ context.Context, key, value string, _ time.Duration) error {
	c.values[key] = value
	return nil
}
func (c *memoryCache) Delete(_ context.Context, keys ...string) error {
	for _, key := range keys {
		delete(c.values, key)
	}
	return nil
}

func request(t *testing.T, router http.Handler, method, path string, body []byte) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(method, path, bytes.NewReader(body)))
	return w
}

func bookFromResponse(t *testing.T, w *httptest.ResponseRecorder) bookstore.Book {
	t.Helper()
	var b bookstore.Book
	if err := json.Unmarshal(w.Body.Bytes(), &b); err != nil {
		t.Fatal(err)
	}
	return b
}

func TestHealthAndReadiness(t *testing.T) {
	store := &memoryStore{books: map[int64]bookstore.Book{1: sampleBook}}
	router := newRouter(store, nil)
	if w := request(t, router, http.MethodGet, "/health", nil); w.Code != 200 {
		t.Fatalf("health: %d %s", w.Code, w.Body.String())
	}
	if w := request(t, router, http.MethodGet, "/ready", nil); w.Code != 200 {
		t.Fatalf("ready: %d %s", w.Code, w.Body.String())
	}
	store.pingErr = errors.New("postgres down")
	if w := request(t, router, http.MethodGet, "/health", nil); w.Code != 200 {
		t.Fatalf("health during outage: %d", w.Code)
	}
	if w := request(t, router, http.MethodGet, "/ready", nil); w.Code != 503 {
		t.Fatalf("ready with postgres down: %d", w.Code)
	}
	store.pingErr = nil
	if w := request(t, newRouter(store, errors.New("redis down")), http.MethodGet, "/ready", nil); w.Code != 503 {
		t.Fatalf("ready with redis down: %d", w.Code)
	}
}

func TestBaselineBookRead(t *testing.T) {
	router := newRouter(&memoryStore{books: map[int64]bookstore.Book{1: sampleBook}}, nil)
	w := request(t, router, http.MethodGet, "/api/books/1", nil)
	if w.Code != 200 || w.Header().Get("X-Cache-Result") != "bypass" {
		t.Fatalf("read: %d %q %s", w.Code, w.Header().Get("X-Cache-Result"), w.Body.String())
	}
	if got := bookFromResponse(t, w); got != sampleBook {
		t.Fatalf("book: %+v", got)
	}
	for _, tc := range []struct {
		path   string
		status int
	}{{"/api/books/999", 404}, {"/api/books/nope", 400}, {"/api/books/0", 400}, {"/api/books/-2", 400}} {
		if w := request(t, router, http.MethodGet, tc.path, nil); w.Code != tc.status {
			t.Errorf("%s: got %d, want %d", tc.path, w.Code, tc.status)
		}
	}
}

func TestSingleflightBookCacheRoundTrip(t *testing.T) {
	store := &memoryStore{books: map[int64]bookstore.Book{1: sampleBook}}
	cache := &memoryCache{values: make(map[string]string)}
	router := httpapi.NewRouter(store, cache)
	path := "/api/labs/singleflight/books/1"

	w := request(t, router, http.MethodGet, path, nil)
	if w.Code != 200 || w.Header().Get("X-Cache-Result") != "miss" || bookFromResponse(t, w) != sampleBook {
		t.Fatalf("miss: %d %q %s", w.Code, w.Header().Get("X-Cache-Result"), w.Body.String())
	}
	var cached bookstore.Book
	if err := json.Unmarshal([]byte(cache.values["singleflight:book:1"]), &cached); err != nil || cached != sampleBook {
		t.Fatalf("cached book: %+v, error: %v", cached, err)
	}

	store.getErr = errors.New("should not load book on cache hit")
	w = request(t, router, http.MethodGet, path, nil)
	if w.Code != 200 || w.Header().Get("X-Cache-Result") != "hit" || bookFromResponse(t, w) != sampleBook {
		t.Fatalf("hit: %d %q %s", w.Code, w.Header().Get("X-Cache-Result"), w.Body.String())
	}
}

func TestStarterLabRoutesBypassCache(t *testing.T) {
	router := newRouter(&memoryStore{books: map[int64]bookstore.Book{1: sampleBook}}, nil)
	paths := []string{
		"/api/labs/warming/books/1",
		"/api/labs/consistency/books/1", "/api/labs/hot-keys/replicated/books/1",
		"/api/labs/hot-keys/local-fallback/books/1", "/api/labs/hot-keys/rate-limited/books/1",
	}
	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			w := request(t, router, http.MethodGet, path, nil)
			if w.Code != 200 || w.Header().Get("X-Cache-Result") != "bypass" {
				t.Fatalf("%d %q %s", w.Code, w.Header().Get("X-Cache-Result"), w.Body.String())
			}
			if got := bookFromResponse(t, w); got != sampleBook {
				t.Fatalf("book: %+v", got)
			}
			if w := request(t, router, http.MethodGet, path[:len(path)-1]+"9", nil); w.Code != 404 {
				t.Fatalf("unknown book: %d", w.Code)
			}
			if w := request(t, router, http.MethodGet, path[:len(path)-1]+"x", nil); w.Code != 400 {
				t.Fatalf("invalid ID: %d", w.Code)
			}
		})
	}
	w := request(t, router, http.MethodPost, "/api/labs/warming", nil)
	if w.Code != http.StatusNotImplemented || !bytes.Contains(w.Body.Bytes(), []byte("incomplete")) {
		t.Fatalf("warming: %d %s", w.Code, w.Body.String())
	}
}

func TestConsistencyUpdatePersistsThroughHTTP(t *testing.T) {
	store := &memoryStore{books: map[int64]bookstore.Book{1: sampleBook}}
	router := newRouter(store, nil)
	body := []byte(`{"title":"Revised Go","author":"Ada","price_cents":4200}`)
	w := request(t, router, http.MethodPut, "/api/labs/consistency/books/1", body)
	if w.Code != 200 {
		t.Fatalf("update: %d %s", w.Code, w.Body.String())
	}
	if got := bookFromResponse(t, w); got.Title != "Revised Go" || got.PriceCents != 4200 {
		t.Fatalf("updated book: %+v", got)
	}
	w = request(t, router, http.MethodGet, "/api/books/1", nil)
	if got := bookFromResponse(t, w); got.Title != "Revised Go" {
		t.Fatalf("subsequent read: %+v", got)
	}
	for _, tc := range []struct {
		path   string
		body   []byte
		status int
	}{
		{"/api/labs/consistency/books/99", body, 404},
		{"/api/labs/consistency/books/x", body, 400},
		{"/api/labs/consistency/books/1", []byte(`{"title":"","author":"Ada","price_cents":1}`), 400},
		{"/api/labs/consistency/books/1", []byte(`{"title":"Good","author":"Ada","price_cents":-1}`), 400},
		{"/api/labs/consistency/books/1", []byte(`{"title":"Good","author":"Ada"}`), 400},
	} {
		if w := request(t, router, http.MethodPut, tc.path, tc.body); w.Code != tc.status {
			t.Errorf("update %s: got %d, want %d: %s", tc.path, w.Code, tc.status, w.Body.String())
		}
	}
}

func TestUnavailableStoreReturns503(t *testing.T) {
	store := &memoryStore{books: map[int64]bookstore.Book{}, getErr: errors.New("db unavailable"), updateErr: errors.New("db unavailable")}
	router := newRouter(store, nil)
	if w := request(t, router, http.MethodGet, "/api/books/1", nil); w.Code != 503 {
		t.Fatalf("read: %d", w.Code)
	}
	if w := request(t, router, http.MethodPut, "/api/labs/consistency/books/1", []byte(`{"title":"Good","author":"Ada","price_cents":1}`)); w.Code != 503 {
		t.Fatalf("update: %d", w.Code)
	}
}
