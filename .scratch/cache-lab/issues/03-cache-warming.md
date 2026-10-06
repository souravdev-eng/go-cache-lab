# 03: Cache warming

**What to build:** A learner can start the stack and see selected seeded books already present in the warming cache namespace before their first request. A lab action repeats the warm operation and reports successes and failures, making the experiment repeatable without restarting the API.

**Blocked by:** 01: Runnable bookstore lab.

**Status:** ready-for-agent

- [ ] Startup warms a known subset of seeded books after PostgreSQL and Redis are ready.
- [ ] The first warmed read identifies a Redis hit; an unwarmed book can still be read through PostgreSQL and then cached.
- [ ] The repeatable warm action reports successful and failed book counts and logs failures.
- [ ] A warm failure does not alter authoritative PostgreSQL book data or prevent unrelated reads.
- [ ] HTTP-level acceptance checks and the guide show how to compare a warmed first read with a cold first read.
