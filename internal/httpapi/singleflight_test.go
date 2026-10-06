package httpapi_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/sauravmajumdar/go-cache-lab/internal/bookstore"
	"github.com/sauravmajumdar/go-cache-lab/internal/httpapi"
)

type observedCache struct {
	mu     sync.Mutex
	values map[string]string
	ttls   map[string]time.Duration
	gets   chan string
}

func newObservedCache() *observedCache {
	return &observedCache{values: make(map[string]string), ttls: make(map[string]time.Duration), gets: make(chan string, 100)}
}
func (c *observedCache) Ping(context.Context) error { return nil }
func (c *observedCache) Get(_ context.Context, key string) (string, error) {
	c.mu.Lock()
	value, ok := c.values[key]
	c.mu.Unlock()
	c.gets <- key
	if !ok {
		return "", errors.New("cache miss")
	}
	return value, nil
}
func (c *observedCache) Set(_ context.Context, key, value string, ttl time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.values[key], c.ttls[key] = value, ttl
	return nil
}
func (c *observedCache) Delete(_ context.Context, keys ...string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, key := range keys {
		delete(c.values, key)
	}
	return nil
}

type observedStore struct {
	mu      sync.Mutex
	loads   map[int64]int
	started chan int64
	release chan struct{}
}

func (s *observedStore) Get(_ context.Context, id int64) (bookstore.Book, error) {
	s.mu.Lock()
	s.loads[id]++
	s.mu.Unlock()
	s.started <- id
	if id == 1 && s.release != nil {
		<-s.release
	}
	if id == 1 {
		return sampleBook, nil
	}
	if id == 2 {
		book := sampleBook
		book.ID = 2
		return book, nil
	}
	return bookstore.Book{}, bookstore.ErrNotFound
}
func (s *observedStore) Update(context.Context, int64, bookstore.BookUpdate) (bookstore.Book, error) {
	return bookstore.Book{}, errors.New("unused")
}
func (s *observedStore) Ping(context.Context) error { return nil }
func (s *observedStore) count(id int64) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.loads[id]
}

func TestSingleflightConcurrentReadsAndTTL(t *testing.T) {
	store := &observedStore{loads: make(map[int64]int), started: make(chan int64, 10), release: make(chan struct{})}
	cache := newObservedCache()
	router := httpapi.NewRouter(store, cache, 17*time.Second)
	const path = "/api/labs/singleflight/books/1"
	responses := make(chan *httptest.ResponseRecorder, 12)
	go func() { responses <- request(t, router, http.MethodGet, path, nil) }()
	if id := <-store.started; id != 1 {
		t.Fatalf("first load: %d", id)
	}
	for i := 0; i < 8; i++ {
		go func() { responses <- request(t, router, http.MethodGet, path, nil) }()
	}
	// Each follower has observed the cold cache before the first load completes.
	for i := 0; i < 10; i++ {
		if key := <-cache.gets; key != "singleflight:book:1" {
			t.Fatalf("cache key: %s", key)
		}
	}
	// Another book must complete while book 1 is blocked in PostgreSQL.
	other := request(t, router, http.MethodGet, "/api/labs/singleflight/books/2", nil)
	if other.Code != 200 || bookFromResponse(t, other).ID != 2 {
		t.Fatalf("other book: %d %s", other.Code, other.Body.String())
	}
	close(store.release)
	for i := 0; i < 9; i++ {
		w := <-responses
		if w.Code != 200 || bookFromResponse(t, w) != sampleBook {
			t.Fatalf("coalesced read: %d %s", w.Code, w.Body.String())
		}
	}
	if got := store.count(1); got != 1 {
		t.Fatalf("same-book database loads: %d, want 1", got)
	}
	if got := store.count(2); got != 1 {
		t.Fatalf("other-book database loads: %d, want 1", got)
	}
	cache.mu.Lock()
	ttl := cache.ttls["singleflight:book:1"]
	cache.mu.Unlock()
	if ttl != 17*time.Second {
		t.Fatalf("cache TTL: %s", ttl)
	}
	if w := request(t, router, http.MethodGet, path, nil); w.Code != 200 || w.Header().Get("X-Cache-Result") != "hit" {
		t.Fatalf("warm read: %d %q", w.Code, w.Header().Get("X-Cache-Result"))
	}
	if err := cache.Delete(context.Background(), "singleflight:book:1"); err != nil {
		t.Fatal(err)
	}
	if w := request(t, router, http.MethodGet, path, nil); w.Code != 200 || w.Header().Get("X-Cache-Result") != "miss" {
		t.Fatalf("read after expiry: %d %q", w.Code, w.Header().Get("X-Cache-Result"))
	}
	if got := store.count(1); got != 2 {
		t.Fatalf("loads after expiry: %d, want 2", got)
	}
}

// The second request saw a stale miss before the first request filled Redis.
// Its second cache check must prevent another PostgreSQL read.
type staleReadCache struct {
	*observedCache
	mu      sync.Mutex
	reads   int
	stale   chan struct{}
	release chan struct{}
}

func (c *staleReadCache) Get(ctx context.Context, key string) (string, error) {
	c.mu.Lock()
	c.reads++
	read := c.reads
	c.mu.Unlock()
	if read == 3 {
		close(c.stale)
		<-c.release
		return "", errors.New("stale cache miss")
	}
	return c.observedCache.Get(ctx, key)
}

func TestSingleflightRechecksCacheAfterStaleMiss(t *testing.T) {
	store := &observedStore{loads: make(map[int64]int), started: make(chan int64, 10), release: make(chan struct{})}
	cache := &staleReadCache{observedCache: newObservedCache(), stale: make(chan struct{}), release: make(chan struct{})}
	router := httpapi.NewRouter(store, cache)
	const path = "/api/labs/singleflight/books/1"
	first := make(chan *httptest.ResponseRecorder, 1)
	go func() { first <- request(t, router, http.MethodGet, path, nil) }()
	<-store.started
	second := make(chan *httptest.ResponseRecorder, 1)
	go func() { second <- request(t, router, http.MethodGet, path, nil) }()
	<-cache.stale
	close(store.release)
	if w := <-first; w.Code != 200 || w.Header().Get("X-Cache-Result") != "miss" {
		t.Fatalf("first read: %d %q", w.Code, w.Header().Get("X-Cache-Result"))
	}
	close(cache.release)
	if w := <-second; w.Code != 200 || w.Header().Get("X-Cache-Result") != "hit" {
		t.Fatalf("stale miss read: %d %q", w.Code, w.Header().Get("X-Cache-Result"))
	}
	if got := store.count(1); got != 1 {
		t.Fatalf("database loads after stale miss: %d, want 1", got)
	}
}

func TestSingleflightUnknownBookIsNotCached(t *testing.T) {
	store := &observedStore{loads: make(map[int64]int), started: make(chan int64, 10)}
	cache := newObservedCache()
	router := httpapi.NewRouter(store, cache)
	for i := 0; i < 2; i++ {
		if w := request(t, router, http.MethodGet, "/api/labs/singleflight/books/999", nil); w.Code != 404 {
			t.Fatalf("unknown book: %d %s", w.Code, w.Body.String())
		}
	}
	if got := store.count(999); got != 2 {
		t.Fatalf("unknown book loads: %d, want 2", got)
	}
	cache.mu.Lock()
	_, cached := cache.values["singleflight:book:999"]
	cache.mu.Unlock()
	if cached {
		t.Fatal("unknown book was cached")
	}
}
