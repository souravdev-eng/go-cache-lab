# Bookstore cache lab

A local Go/Gin bookstore API backed by PostgreSQL, with Redis ready for six cache exercises. Request coalescing is implemented; the other lab reads still load from PostgreSQL and return `X-Cache-Result: bypass`.

Start with the [cache-pattern code tour](docs/code-tour.md). It follows one request and shows which file to edit first.

## Start and reset

Run everything in containers:

```sh
docker compose up --build
```

The API is at `http://localhost:8080`. Compose starts one PostgreSQL instance and one Redis instance, waits for both health checks, then starts the API. The API creates the `books` table and inserts fixed seed rows on startup. Rerunning setup does not duplicate or overwrite books. Book **1** is the designated popular book for the hot-key exercises.

To run Go on the host with automatic reload while keeping PostgreSQL and Redis in containers, install [Air](https://github.com/air-verse/air) once:

```sh
PATH="/usr/local/go/bin:$PATH" go install github.com/air-verse/air@v1.52.3
```

Then use the script from the project root:

```sh
./dev.sh start
./dev.sh status
./dev.sh stop
```

The script runs Air in the background. Saving a `.go` file rebuilds and restarts the API automatically; its output is in `tmp/dev.log`. The script stops the Compose API to keep port 8080 free. To go back to the containerized API, run `./dev.sh stop` followed by `docker compose up -d --build api`.

The sample configuration uses host ports 5433 for PostgreSQL and 6380 for Redis. `CACHE_TTL` controls the singleflight Redis lifetime (default `30s`); Compose accepts the same variable. The Docker Compose API uses service names and internal ports instead. This is a local learning project with no authentication.

Use Go 1.22 or newer for host startup. On the macOS 26 development machine used for this scaffold, the Homebrew Go 1.22.1 binary fails to launch tests; `/usr/local/go/bin/go` works. If your system Go already works, the `PATH` prefix above is optional.

When using `dev.sh`, use `./dev.sh stop` to stop both the host API and its dependencies. For the all-Docker workflow, stop the stack with `docker compose down`. To erase database and Redis state and return to the original seed data, run `docker compose down -v`, then `docker compose up --build`.

## Start with one cache problem

```sh
curl -i localhost:8080/api/books/1
curl -i localhost:8080/api/labs/singleflight/books/1
```

Book 1 is the designated popular book. The first URL is the database-only baseline and returns `X-Cache-Result: bypass`. The second uses Redis: a cold read returns `miss`, another concurrent cold request can return `shared`, and a warm read returns `hit`. Open `internal/httpapi/singleflight.go` to follow the flow.

Clear the exercise key, send 20 simultaneous requests for the same book, then inspect the API's database-read logs and Redis key lifetime:

```sh
docker compose exec redis redis-cli DEL singleflight:book:1
seq 1 20 | xargs -P20 -I{} curl -s -D - -o /dev/null localhost:8080/api/labs/singleflight/books/1 | grep -i '^x-cache-result:'
docker compose logs --since=1m api | grep '"route":"/api/labs/singleflight/books/:id"' | grep '"source":"postgres"'
docker compose exec redis redis-cli TTL singleflight:book:1
curl -i localhost:8080/api/labs/singleflight/books/1
```

Expect one PostgreSQL load for a same-book cold burst handled by one API process, plus `shared` or `hit` outcomes for the other requests. The final curl should be a `hit` until the TTL expires; after expiry, the next read is a `miss` and loads PostgreSQL again. The Redis TTL command shows remaining seconds (`-2` means the key has expired). Different book IDs use different coalescing keys and can load independently. This exercise does not coordinate requests across API processes.

| Cache pattern | File to edit | Routes for the exercise |
| --- | --- | --- |
| Request coalescing | `singleflight.go` | `GET /api/labs/singleflight/books/:id` |
| Warming | `warming.go` | `GET /api/labs/warming/books/:id`, `POST /api/labs/warming` |
| Consistency | `consistency.go` | `GET` and `PUT /api/labs/consistency/books/:id` |
| Replicated hot key | `replicated.go` | `GET /api/labs/hot-keys/replicated/books/:id` |
| Local fallback | `local_fallback.go` | `GET /api/labs/hot-keys/local-fallback/books/:id` |
| Rate limiting | `rate_limited.go` | `GET /api/labs/hot-keys/rate-limited/books/:id` |

These files are in `internal/httpapi/`. `book_get.go` is the PostgreSQL baseline; `status.go` contains `GET /health` and `GET /ready`. The URL map is in `router.go`, SQL is in `internal/bookstore/postgres.go`, and the Redis client is in `internal/cache/redis.go`. You can return to those files when a pattern needs them.

For other starter checks, `GET /api/books/999` returns 404 and `GET /api/books/not-a-number` returns 400. `POST /api/labs/warming` returns 501 until that exercise is done. A valid `PUT /api/labs/consistency/books/1` writes the update to PostgreSQL. Readiness reports 503 if PostgreSQL or Redis is unavailable.

## Exercise TODOs

The remaining starter read handlers deliberately use PostgreSQL. Replace one pattern's handler as you complete its issue in `.scratch/cache-lab/issues/`. Use a distinct Redis prefix per exercise and expose outcomes in `X-Cache-Result` and structured logs. Suggested prefixes are `singleflight:book:`, `warming:book:`, `consistency:book:`, `hot-keys:replicated:book:`, `hot-keys:local-fallback:book:`, and `hot-keys:rate-limited:book:`. Keep 404s uncached. `consistency.go` has the TODO to invalidate cache copies after its database write.

- **Request coalescing (complete):** Cache aside with a per-book, in-process singleflight group. Redis is rechecked inside the group. Same-book cold reads share one database load; other book IDs proceed independently. Coalescing does not span API processes.
- **Warming:** On startup and on `POST /api/labs/warming`, load selected seed books into the warming namespace. Report success and failure counts. A warmed first read should hit Redis; a failed warm should be visible.
- **Consistency:** Cache aside on reads. After a successful PostgreSQL update, invalidate every Redis namespace, each logical replica, and the local copy. Test read-after-update across routes. Concurrent reads and writes can still race; this lab does not claim strict consistency.
- **Replicated hot key:** Store several logical copies of popular book 1 and show the selected copy in the response or logs. Multiple keys in one Redis instance demonstrate copy management, not load spreading across Redis nodes.
- **Local fallback:** Keep a bounded in-process cache with a shorter TTL than Redis. During a Redis outage, serve a fresh local copy; otherwise read PostgreSQL and refill it. This fallback is per API process.
- **Rate limiting:** Apply a configurable one-process, per-book threshold before the expensive read. A rejected burst should return 429 with `Retry-After`; a distributed limiter would need shared state.

Inspect future Redis keys with `docker compose exec redis redis-cli --scan --pattern '*book*'`; clear a particular exercise key with `docker compose exec redis redis-cli DEL singleflight:book:1`. To try a Redis outage after implementing fallback, run `docker compose stop redis`, read a primed book through the local-fallback route, then run `docker compose start redis`.

Run the tests with `go test ./...`. To also check seeding and updates through the HTTP API against disposable PostgreSQL and Redis containers, run `CACHE_LAB_INTEGRATION=1 go test ./...` with Docker running. The request-coalescing acceptance checks run in the ordinary test suite; the remaining exercise scenarios are future checks.
