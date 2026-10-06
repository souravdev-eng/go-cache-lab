# Bookstore cache lab

A local Go/Gin bookstore API backed by PostgreSQL, with Redis ready for six cache exercises. The starter API works before any exercise is completed. All lab reads currently load from PostgreSQL and return `X-Cache-Result: bypass`.

## Start and reset

Run everything in containers:

```sh
docker compose up --build
```

The API is at `http://localhost:8080`. Compose starts one PostgreSQL instance and one Redis instance, waits for both health checks, then starts the API. The API creates the `books` table and inserts fixed seed rows on startup. Rerunning setup does not duplicate or overwrite books. Book **1** is the designated popular book for the hot-key exercises.

To run Go on the host while keeping the dependencies in containers:

```sh
docker compose up -d postgres redis
set -a; . ./.env.example; set +a
go run ./cmd/api
```

The sample configuration uses host ports 5433 for PostgreSQL and 6380 for Redis. Its cache settings are exercise inputs; the starter routes do not use them yet. The Docker Compose API uses service names and internal ports instead. This is a local learning project with no authentication.

Use Go 1.22 or newer for host startup. On the macOS 26 development machine used for this scaffold, the Homebrew Go 1.22.1 binary fails to launch tests; `/usr/local/go/bin/go` works.

Stop the stack with `docker compose down`. To erase database and Redis state and return to the original seed data, run `docker compose down -v`, then `docker compose up --build`.

## Try the starter API

```sh
curl -i localhost:8080/health
curl -i localhost:8080/ready
curl -i localhost:8080/api/books/1
curl -i localhost:8080/api/books/999
curl -i localhost:8080/api/books/not-a-number
curl -i localhost:8080/api/labs/singleflight/books/1
curl -i -X POST localhost:8080/api/labs/warming
curl -i -X PUT localhost:8080/api/labs/consistency/books/1 \
  -H 'Content-Type: application/json' \
  -d '{"title":"The Go Programming Language, revised","author":"Alan A. A. Donovan and Brian W. Kernighan","price_cents":4099}'
curl -i localhost:8080/api/books/1
```

`/health` reports that the process is responding. `/ready` checks PostgreSQL and Redis and returns 503 when either is unavailable. Unknown books return 404, invalid IDs and update bodies return 400, and unavailable database reads return 503. Each successful read returns the same `id`, `title`, `author`, `price_cents`, and `updated_at` fields. The update writes through to PostgreSQL.

Other starter GET routes (replace `1` with a book ID):

| Exercise | Route |
| --- | --- |
| Cache warming | `/api/labs/warming/books/1` |
| Cache consistency | `/api/labs/consistency/books/1` |
| Replicated hot key | `/api/labs/hot-keys/replicated/books/1` |
| Local fallback | `/api/labs/hot-keys/local-fallback/books/1` |
| Rate limiting | `/api/labs/hot-keys/rate-limited/books/1` |

`POST /api/labs/warming` returns 501 with `status: incomplete` until the warming exercise is implemented. The lab routes are independent paths so their future Redis key namespaces can be kept separate.

## Exercise TODOs

The starter read handlers in `internal/bookstore/http.go` deliberately use PostgreSQL. Replace each route's read path as you complete its issue in `.scratch/cache-lab/issues/`. Use a distinct Redis prefix per exercise and expose outcomes in `X-Cache-Result` and structured logs. Suggested prefixes are `singleflight:book:`, `warming:book:`, `consistency:book:`, `hot-keys:replicated:book:`, `hot-keys:local-fallback:book:`, and `hot-keys:rate-limited:book:`. Keep 404s uncached. The starter update has a TODO to invalidate cache copies after its database write.

- **Request coalescing:** Cache aside with a per-book, in-process singleflight group. Recheck Redis inside the group. Same-book cold reads should share one database load; other book IDs should proceed independently. Coalescing does not span API processes.
- **Warming:** On startup and on `POST /api/labs/warming`, load selected seed books into the warming namespace. Report success and failure counts. A warmed first read should hit Redis; a failed warm should be visible.
- **Consistency:** Cache aside on reads. After a successful PostgreSQL update, invalidate every Redis namespace, each logical replica, and the local copy. Test read-after-update across routes. Concurrent reads and writes can still race; this lab does not claim strict consistency.
- **Replicated hot key:** Store several logical copies of popular book 1 and show the selected copy in the response or logs. Multiple keys in one Redis instance demonstrate copy management, not load spreading across Redis nodes.
- **Local fallback:** Keep a bounded in-process cache with a shorter TTL than Redis. During a Redis outage, serve a fresh local copy; otherwise read PostgreSQL and refill it. This fallback is per API process.
- **Rate limiting:** Apply a configurable one-process, per-book threshold before the expensive read. A rejected burst should return 429 with `Retry-After`; a distributed limiter would need shared state.

For concurrent traffic after implementing coalescing, try `seq 1 20 | xargs -P20 -I{} curl -s -o /dev/null localhost:8080/api/labs/singleflight/books/1`. Inspect future Redis keys with `docker compose exec redis redis-cli --scan --pattern '*book*'`; clear a particular exercise key with `docker compose exec redis redis-cli DEL singleflight:book:1`. To try a Redis outage after implementing fallback, run `docker compose stop redis`, read a primed book through the local-fallback route, then run `docker compose start redis`.

Run the starter tests with `go test ./...`. The unfinished exercise acceptance scenarios above are future checks; ordinary tests verify the working starter HTTP behavior.
