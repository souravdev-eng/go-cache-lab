# Follow one cache pattern

Start with request coalescing. The route now uses Redis and shares simultaneous cold loads within one API process. You do not need to understand the entire server first.

## Read these files in order

1. `internal/httpapi/book_get.go` is the PostgreSQL-only comparison route.
2. `internal/httpapi/singleflight.go` contains the completed request-coalescing exercise.
3. `internal/bookstore/postgres.go` shows the SQL behind `a.store.Get`.
4. Open `internal/httpapi/router.go` only if you need to see how a URL reaches a handler. Open `cmd/api/main.go` only if you need to see how PostgreSQL and Redis are connected at startup.

The request path is:

```text
curl -> Gin router -> singleflightGet -> Redis cache
                    | cold miss -> per-book flight -> PostgreSQL -> Redis fill
                    <- JSON book and X-Cache-Result: miss/shared/hit <-
```

Redis is connected at startup. The singleflight GET route uses it; the remaining starter GET routes are still PostgreSQL-backed exercises.

## Follow one read

Try `curl -i localhost:8080/api/labs/singleflight/books/1` after starting the stack. In `singleflight.go`:

1. `bookID(c)` reads `:id` from the URL. It returns a positive `int64` or sends a 400 response. The small shared function is in `http_helpers.go`.
2. `a.cachedBook` asks Redis for the book. A warm read returns `hit` without using PostgreSQL.
3. On a miss, `a.doBookLoad` joins or starts a per-book flight. The leader rechecks Redis, asks PostgreSQL on a second miss, and fills Redis with `CACHE_TTL`. Concurrent followers share its result.
4. `storeError` sends 404 when the book does not exist and 503 when PostgreSQL cannot serve the request. Missing books are not cached.
5. `c.Header` reports `miss`, `shared`, or `hit`; `c.JSON` sends the book fields.

Send a cold burst using the commands in the [README](../README.md). Observe one PostgreSQL load per book within this API process. Redis methods live in `internal/cache/redis.go`.

## Go syntax when you need it

| Code | Meaning |
| --- | --- |
| `package httpapi` | Files in the same directory and package can use each other's types and functions. |
| `type Book struct { ... }` | A value with named fields. |
| `func (a *api) singleflightGet(...)` | A method on `api`; `a.store` and `a.cache` are available inside it. |
| `type BookStore interface { ... }` | A list of methods the HTTP code needs. `PostgresStore` satisfies it by having those methods; Go needs no `implements` declaration. |
| `book, err := a.store.Get(...)` | A Go function can return both a result and an error. Check `err` before using `book`. |
| `context.Context` | Carries cancellation and time limits. Passing the request context lets a database query stop if the request is cancelled. |
| `defer store.Close()` | Runs `Close` when the surrounding function returns. |

`internal/` is a Go convention: packages there are used by this project, not imported by unrelated projects. The `cmd/api/` folder contains the executable program.

## Tests and the next step

`internal/httpapi/http_test.go` sends requests to the router without starting a network server. It uses an in-memory book store so `go test ./...` works without Docker. `integration_test.go` uses disposable PostgreSQL and Redis containers; run it with `CACHE_LAB_INTEGRATION=1 go test ./...` when Docker is running.

Compare `book_get.go` and `singleflight.go`, clear the Redis key, run a concurrent burst, then run the tests. The pattern table in the [README](../README.md) shows where every other exercise begins.
