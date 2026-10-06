# Code tour for a Go beginner

You do not need to understand every file before starting an exercise. Read one request from the URL to the database, then return to the other files when you need them.

## Read these files in order

1. `internal/bookstore/book.go` defines a `Book` and the fields returned as JSON. The text inside each `json:"..."` tag is the field name in an HTTP response.
2. `internal/httpapi/http.go` lists every URL and the function that handles it. For example, `r.GET("/api/labs/singleflight/books/:id", a.singleflightGet)` sends that URL to `singleflightGet`.
3. `internal/httpapi/singleflight_get.go` shows the complete starter read for one exercise. The other GET files follow the same shape so you can change one without changing the others.
4. `internal/bookstore/postgres.go` shows the SQL used by `a.store.Get` and `a.store.Update`.
5. `cmd/api/main.go` creates the real PostgreSQL store and Redis cache, initializes the database, and passes both to the HTTP router.

The request path is:

```text
curl -> Gin router -> singleflightGet -> PostgresStore.Get -> PostgreSQL
                    <- JSON book and X-Cache-Result: bypass <-
```

Redis is connected at startup, but the starter GET routes do not read or write it yet. That is the work in the exercise TODOs.

## Follow one read

Try `curl -i localhost:8080/api/labs/singleflight/books/1` after starting the stack. In `singleflight_get.go`:

1. `bookID(c)` reads `:id` from the URL. It returns a positive `int64` or sends a 400 response. The small shared function is in `http_helpers.go`.
2. `a.store.Get(c.Request.Context(), id)` asks PostgreSQL for that book. `a.store` is the database object passed in by `main.go`.
3. `if err != nil` handles a failed read. `storeError` sends 404 when the book does not exist and 503 when PostgreSQL cannot serve the request.
4. `c.Header` marks this starter read as a cache bypass. `c.JSON` sends status 200 and the book fields as JSON.

The `TODO(singleflight)` belongs to this route. When you implement it, edit this file and use `a.cache.Get` and `a.cache.Set` here. The Redis methods live in `internal/cache/redis.go`.

## Go syntax you will see here

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

Start with the baseline `book_get.go` and the exercise file `singleflight_get.go`. Compare their response to the same book. Then work through one TODO and run the tests again. The route-to-file table in the [README](../README.md) shows where every other exercise begins.
