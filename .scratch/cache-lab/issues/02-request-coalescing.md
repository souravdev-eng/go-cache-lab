# 02: Request coalescing

**What to build:** A learner can send many simultaneous cold requests for the same book to the singleflight lab route and observe one PostgreSQL load shared by those requests. Requests for different books remain independent. Subsequent reads come from Redis until the configurable TTL expires. The exercise guide shows how to clear the relevant key, generate concurrent traffic, and inspect the result.

**Blocked by:** 01: Runnable bookstore lab.

**Status:** ready-for-agent

- [ ] The first read of a book uses PostgreSQL and populates the singleflight Redis namespace; a later read is an observable cache hit.
- [ ] Concurrent misses for one book share one PostgreSQL load within one API process, including a cache recheck within the coalesced work.
- [ ] Concurrent reads for different book IDs do not wait on the same coalescing key.
- [ ] Unknown books remain 404 and are not cached.
- [ ] HTTP-level acceptance checks cover cold, warm, concurrent same-book, and different-book behavior without relying on elapsed-time thresholds alone.
- [ ] The guide states that this exercise does not coordinate requests across API processes.
