package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sauravmajumdar/go-cache-lab/internal/bookstore"
	"github.com/sauravmajumdar/go-cache-lab/internal/cache"
	"github.com/sauravmajumdar/go-cache-lab/internal/httpapi"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		slog.Error("DATABASE_URL is required")
		os.Exit(1)
	}
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6380"
	}
	httpAddr := os.Getenv("HTTP_ADDR")
	if httpAddr == "" {
		httpAddr = ":8080"
	}
	cacheTTL := 30 * time.Second
	if value := os.Getenv("CACHE_TTL"); value != "" {
		var err error
		cacheTTL, err = time.ParseDuration(value)
		if err != nil || cacheTTL <= 0 {
			slog.Error("CACHE_TTL must be a positive duration", "value", value)
			os.Exit(1)
		}
	}

	store, err := bookstore.OpenPostgres(databaseURL)
	if err != nil {
		slog.Error("open postgres", "error", err)
		os.Exit(1)
	}
	defer store.Close()
	redisCache := cache.NewRedisCache(redisAddr)
	defer redisCache.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := store.Ping(ctx); err != nil {
		slog.Error("postgres unavailable", "error", err)
		os.Exit(1)
	}
	if err := redisCache.Ping(ctx); err != nil {
		slog.Error("redis unavailable", "error", err)
		os.Exit(1)
	}
	if err := store.Initialize(ctx); err != nil {
		slog.Error("initialize database", "error", err)
		os.Exit(1)
	}

	// The router receives the real database and Redis implementations here.
	server := &http.Server{
		Addr:              httpAddr,
		Handler:           httpapi.NewRouter(store, redisCache, cacheTTL),
		ReadHeaderTimeout: 5 * time.Second,
	}
	shutdownCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		<-shutdownCtx.Done()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(ctx)
	}()

	slog.Info("bookstore API listening", "address", httpAddr)

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error("serve HTTP", "error", err)
		os.Exit(1)
	}
}
