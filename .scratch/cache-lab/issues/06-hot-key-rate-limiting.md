# 06: Hot-key rate limiting

**What to build:** A learner can send a burst to the rate-limited hot-key route, receive successful responses up to a configured per-book threshold, then receive 429 responses with `Retry-After`. A different book has its own allowance, and requests succeed again after the window resets.

**Blocked by:** 01: Runnable bookstore lab.

**Status:** ready-for-agent

- [ ] The threshold and window are configurable and small enough for a quick local demonstration.
- [ ] The limit is evaluated before the book read; a request over the limit returns 429 and `Retry-After`.
- [ ] A burst for one book does not consume another book's allowance.
- [ ] Requests become eligible again after the window resets.
- [ ] HTTP-level acceptance checks cover allowed requests, rejection, per-book independence, and retry behavior.
- [ ] The guide identifies this as a one-process limiter and describes the distributed variant as future work.
