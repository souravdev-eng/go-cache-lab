# 05: Local fallback cache

**What to build:** A learner can prime a bounded in-process book cache, make Redis unavailable, and still read an unexpired local copy through the fallback lab route. If no local copy exists, the route reads PostgreSQL and repopulates local state. The response makes the source visible.

**Blocked by:** 01: Runnable bookstore lab.

**Status:** ready-for-agent

- [ ] A successful normal read populates the local cache as well as using the Redis read path.
- [ ] During a Redis error, an unexpired local copy serves the same book with an observable local-fallback outcome.
- [ ] During a Redis error with no local copy, PostgreSQL supplies the book and the local cache is populated.
- [ ] The local cache has a configurable size bound and a TTL shorter than Redis's TTL.
- [ ] Unknown books remain 404; an unavailable PostgreSQL read with no usable cache returns 503.
- [ ] HTTP-level acceptance checks cover Redis outage, local hit, local expiry, and database fallback; the guide explains that the fallback is per API process.
