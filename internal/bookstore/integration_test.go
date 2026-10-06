package bookstore_test

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/sauravmajumdar/go-cache-lab/internal/bookstore"
)

// Run with CACHE_LAB_INTEGRATION=1 go test ./... when Docker is available.
// Each run creates and removes its own PostgreSQL and Redis containers.
func TestSeedAndUpdateThroughHTTPWithDisposableServices(t *testing.T) {
	if os.Getenv("CACHE_LAB_INTEGRATION") != "1" {
		t.Skip("set CACHE_LAB_INTEGRATION=1 to run with disposable PostgreSQL and Redis")
	}
	if _, err := exec.LookPath("docker"); err != nil {
		t.Fatal(err)
	}
	postgresPort := runContainer(t, "postgres:16-alpine", "5432/tcp", "-e", "POSTGRES_USER=bookstore", "-e", "POSTGRES_PASSWORD=bookstore", "-e", "POSTGRES_DB=bookstore")
	redisPort := runContainer(t, "redis:7-alpine", "6379/tcp")
	store, err := bookstore.OpenPostgres(fmt.Sprintf("postgres://bookstore:bookstore@127.0.0.1:%s/bookstore?sslmode=disable", postgresPort))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	cache := bookstore.NewRedisCache("127.0.0.1:" + redisPort)
	t.Cleanup(func() { _ = cache.Close() })
	deadline := time.Now().Add(30 * time.Second)
	for {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		postgresErr, redisErr := store.Ping(ctx), cache.Ping(ctx)
		cancel()
		if postgresErr == nil && redisErr == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("dependencies not ready: postgres=%v redis=%v", postgresErr, redisErr)
		}
		time.Sleep(200 * time.Millisecond)
	}
	ctx := context.Background()
	if err := store.Initialize(ctx); err != nil {
		t.Fatal(err)
	}
	if err := store.Initialize(ctx); err != nil {
		t.Fatalf("repeat initialization: %v", err)
	}
	router := bookstore.NewRouter(store, cache)
	if w := request(t, router, http.MethodGet, "/ready", nil); w.Code != 200 {
		t.Fatalf("ready: %d %s", w.Code, w.Body.String())
	}
	for _, tc := range []struct{ id, title string }{
		{"1", "The Go Programming Language"},
		{"2", "Designing Data-Intensive Applications"},
		{"3", "Database Internals"},
	} {
		w := request(t, router, http.MethodGet, "/api/books/"+tc.id, nil)
		if w.Code != 200 || w.Header().Get("X-Cache-Result") != "bypass" {
			t.Fatalf("seed %s: %d %s", tc.id, w.Code, w.Body.String())
		}
		if got := bookFromResponse(t, w); got.Title != tc.title {
			t.Fatalf("seed %s title: %q", tc.id, got.Title)
		}
	}
	update := []byte(`{"title":"Revised Go","author":"Ada","price_cents":4200}`)
	w := request(t, router, http.MethodPut, "/api/labs/consistency/books/1", update)
	if w.Code != 200 || bookFromResponse(t, w).Title != "Revised Go" {
		t.Fatalf("update: %d %s", w.Code, w.Body.String())
	}
	if err := store.Initialize(ctx); err != nil {
		t.Fatal(err)
	}
	w = request(t, router, http.MethodGet, "/api/books/1", nil)
	if w.Code != 200 || bookFromResponse(t, w).Title != "Revised Go" {
		t.Fatalf("read after update and repeat setup: %d %s", w.Code, w.Body.String())
	}
}

func runContainer(t *testing.T, image, containerPort string, args ...string) string {
	t.Helper()
	command := append([]string{"run", "-d", "--rm", "-P"}, args...)
	command = append(command, image)
	out, err := exec.Command("docker", command...).CombinedOutput()
	if err != nil {
		t.Fatalf("start %s: %v: %s", image, err, out)
	}
	id := strings.TrimSpace(string(out))
	t.Cleanup(func() { _ = exec.Command("docker", "rm", "-f", id).Run() })
	out, err = exec.Command("docker", "port", id, containerPort).CombinedOutput()
	if err != nil {
		t.Fatalf("port for %s: %v: %s", image, err, out)
	}
	line := strings.SplitN(strings.TrimSpace(string(out)), "\n", 2)[0]
	_, port, err := net.SplitHostPort(line)
	if err != nil {
		t.Fatalf("parse docker port %q: %v", line, err)
	}
	return port
}
