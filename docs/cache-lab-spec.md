# Cache Lab: Go, PostgreSQL, Redis, and Gin

## Problem Statement

I am studying cache behavior in system design and need a small, runnable project where I can change code, send concurrent requests, and observe the result. Starting from an empty Go repository means spending learning time on containers, database setup, and HTTP plumbing. A finished implementation of every cache pattern would remove the hands-on work I want to do myself.

## Solution

Build a bookstore REST API learning scaffold with one PostgreSQL database, Redis as the cache layer, and a Gin server. The project starts with working infrastructure, seeded books, a database-backed baseline endpoint, and separate routes for each cache exercise. Each exercise has a clear expected behavior, observable response and log signals, and focused TODOs for the learner to implement the pattern. The scaffold must run before those TODOs are completed. The completed exercises should demonstrate request coalescing, cache warming, cache consistency, hot-key replication, local fallback caching, and rate limiting.

## User Stories

1. As a learner, I want one command to start the API, PostgreSQL, and Redis, so that I can begin experimenting quickly.
2. As a learner, I want a sample environment configuration, so that I can see and change connection details and cache settings.
3. As a learner, I want deterministic seed books, so that I can repeat the same experiments.
4. As a learner, I want a health endpoint, so that I can tell whether the API process is running.
5. As a learner, I want a readiness endpoint, so that I can tell whether PostgreSQL and Redis are reachable.
6. As a learner, I want a database-only book read endpoint, so that I have a baseline for comparing cached reads.
7. As a learner, I want a clear 404 response for an unknown book, so that I can distinguish missing data from a cache problem.
8. As a learner, I want separate routes and cache key namespaces for each exercise, so that one experiment does not hide the behavior of another.
9. As a learner, I want cache result information in development responses and logs, so that I can see a miss, hit, coalesced request, or fallback.
10. As a learner, I want to create simultaneous requests for an uncached book, so that I can reproduce a cache stampede.
11. As a learner, I want to add request coalescing to one endpoint, so that concurrent misses for the same book share one database load in one API process.
12. As a learner, I want requests for different books to proceed independently, so that coalescing does not serialize unrelated traffic.
13. As a learner, I want the warmed endpoint to have selected books loaded into Redis at startup, so that I can compare a warm first read with a cold first read.
14. As a learner, I want to trigger warming again through a lab route, so that I can repeat the experiment without restarting the stack.
15. As a learner, I want warming failures to be visible, so that I know whether a supposedly warm read is actually cold.
16. As a learner, I want to update a book through the consistency lab, so that I can observe what happens to previously cached data.
17. As a learner, I want a read after a successful update to return the new book data, so that I can verify cache invalidation.
18. As a learner, I want all cached copies of an updated book to be invalidated or refreshed, so that a different lab route does not silently serve the old value.
19. As a learner, I want cache entries to expire, so that I can observe TTL as a second bound on stale data.
20. As a learner, I want one hot-key route to choose among several logical Redis copies of a popular book, so that I can study read distribution across keys.
21. As a learner, I want a book update to affect every logical hot-key copy, so that replication does not create conflicting answers.
22. As a learner, I want a small in-process cache on the fallback route, so that reads can continue when Redis is unavailable and a fresh local copy exists.
23. As a learner, I want the fallback route to use PostgreSQL when neither Redis nor the local cache has the book, so that a Redis outage does not automatically make every read fail.
24. As a learner, I want local entries to have a short TTL and size bound, so that I can observe the freshness and memory tradeoff.
25. As a learner, I want a per-book request limit on one hot-key route, so that excessive demand returns an explicit 429 response.
26. As a learner, I want a Retry-After value on limited responses, so that I can tell when to retry.
27. As a learner, I want a short exercise guide with commands and expected observations, so that I can implement each pattern myself.
28. As a learner, I want the ordinary test command to pass on the initial scaffold, so that incomplete exercises do not make the starter project look broken.
29. As a learner, I want acceptance checks for each completed exercise, so that I can verify behavior after filling in its TODOs.

## Implementation Decisions

- The deliverable is an educational scaffold, not a finished implementation of every cache pattern. Infrastructure, database access, seed data, Gin routing, the baseline read, and shared configuration work from the start. Pattern-specific logic remains as clearly marked exercises with behavior and verification instructions. An incomplete exercise must not prevent the API or unrelated exercises from running.
- In the initial scaffold, each lab GET route can return the book through the working PostgreSQL read path with a `bypass` cache outcome. The consistency update writes to PostgreSQL. The warming action can report that its exercise is incomplete. The learner then replaces each route's focused TODO with the specified pattern behavior.
- Use a single `books` domain with `id`, `title`, `author`, `price_cents`, and `updated_at`. PostgreSQL is authoritative. Seed a small, fixed set of books, including at least one designated popular book. Repeated setup must not duplicate seed rows.
- Docker Compose starts one PostgreSQL service, one Redis service, and the Go API. Dependency readiness, schema initialization, and seed setup are automated. The Go application can also run on the host using the sample configuration. No authentication is required; document that this is a local learning project.
- Keep one small application composition point that wires the book store, cache access, lab handlers, and router. Avoid adding a general cache framework or separate services for each pattern.
- Use separate routes for the experiments, all returning the same book JSON shape on success:
  - `GET /api/books/:id`: direct PostgreSQL baseline.
  - `GET /api/labs/singleflight/books/:id`: cache-aside read with a request-coalescing exercise.
  - `GET /api/labs/warming/books/:id` and `POST /api/labs/warming`: warmed read and repeatable warm action.
  - `GET /api/labs/consistency/books/:id` and `PUT /api/labs/consistency/books/:id`: cached read and book update exercise.
  - `GET /api/labs/hot-keys/replicated/books/:id`: logical hot-key replica exercise.
  - `GET /api/labs/hot-keys/local-fallback/books/:id`: local fallback exercise.
  - `GET /api/labs/hot-keys/rate-limited/books/:id`: per-book rate-limit exercise.
  - `GET /health` and `GET /ready`: process and dependency checks.
- A successful book read returns 200; an unknown ID returns 404; invalid input returns 400; unavailable dependencies that cannot be bypassed return 503. A valid consistency update returns the updated book. The rate-limited route returns 429 and `Retry-After` when its configured per-book threshold is exceeded.
- The development API exposes a simple cache outcome header, such as `X-Cache-Result`, and structured logs for database loads, Redis reads, warm attempts, coalescing, local fallback, and rejected requests. These are learning aids, not a production metrics system.
- Give each lab a distinct Redis key prefix. Set short, configurable TTLs. Avoid caching 404 responses in the initial exercises, which keeps missing-book behavior simple.
- Request coalescing is per book ID and within one API process. A leader checks Redis, loads PostgreSQL on a miss, and fills Redis; concurrent followers share that load. A second cache check inside the coalesced operation prevents an unnecessary database read. The guide explains that this does not coordinate across API instances.
- Cache warming loads a known subset of seeded books into the warming namespace after dependencies are ready. The repeatable warm action reports how many books succeeded and failed. A failed warm attempt is reported and logged; it does not corrupt PostgreSQL data.
- Cache consistency uses cache-aside reads and invalidation after a successful database commit. A failed database update does not invalidate or pretend to succeed. The update path also removes or refreshes any cached copy of that book in other exercise namespaces, including logical hot-key copies and the in-process cache. Document the remaining race between concurrent reads and writes as a follow-up experiment rather than promising strict consistency.
- Hot-key replication uses a small configurable number of logical keys in the single Redis instance and distributes reads among them. The guide explicitly states that this demonstrates key-copy management and read selection but does not provide node-level load spreading; that would require multiple Redis nodes or a cluster.
- The local fallback route uses a bounded in-process cache with a shorter TTL than Redis. During a Redis error it serves an unexpired local value; otherwise it reads PostgreSQL and repopulates the local cache. The response identifies local fallback. Book updates evict the local copy. The guide explains that this fallback is per API process.
- The rate-limited route applies a small, configurable per-book threshold before the expensive read. It uses a simple one-process limiter appropriate to this single-API lab. The guide describes how a distributed limiter would differ.
- The README gives setup and teardown commands, endpoint examples, a way to generate concurrent requests, a way to inspect Redis keys, a Redis outage experiment, and one observation checklist per pattern. It explains which TODOs the learner completes and what working behavior to expect.

## Testing Decisions

- The primary test seam is the HTTP API with a running Gin router and disposable PostgreSQL and Redis services. Tests assert status, response body, relevant headers, and externally visible data behavior. They do not assert helper calls, private types, or implementation-specific function names.
- The initial scaffold has passing checks for startup, health, readiness, deterministic seed data, the database baseline, invalid IDs, and unknown books. There is no existing test prior art in this repository; use ordinary Go tests and keep the test setup small.
- Each exercise has an acceptance scenario in the guide. After the learner implements it, behavior checks should cover: one database load for concurrent same-book cold reads; independent progress for different books; warmed first read; repeatable warming; read-after-update freshness; invalidation of all logical copies; a local read during Redis outage; expiry and database fallback; and 429 with Retry-After under a burst.
- Prefer deterministic concurrency controls and observable counters or logs over timing-only assertions. A test should fail because the visible behavior is wrong, not because a machine is slow.
- Exercise acceptance checks may be opt-in while TODOs remain. The default test command stays green for the runnable starter scaffold.

## Out of Scope

- Authentication, authorization, user accounts, and a production deployment.
- A complete solution to every exercise in the initial scaffold.
- Redis Cluster, multiple Redis nodes, cross-process singleflight, and distributed rate limiting.
- Strict transactional consistency between PostgreSQL and Redis under concurrent reads and writes.
- A generic cache framework, event bus, message queue, or separate microservices.
- Production-grade load testing and monitoring infrastructure.

## Further Notes

- The current repository contains only a README and has no established domain glossary, ADRs, code conventions, or tests to preserve.
- Keep comments and exercises focused on what the learner can change and observe. Small deliberate TODOs are more useful here than a polished abstraction that hides the cache behavior.
- The proposed single test seam is the HTTP boundary; dependency failures can be exercised by controlling disposable services during tests. Confirm this seam before publishing the issue.
