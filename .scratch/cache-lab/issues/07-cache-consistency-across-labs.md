# 07: Cache consistency across labs

**What to build:** A learner can prime every cache lab with one book, update that book through the consistency route, and immediately read the new value through each route. The update commits to PostgreSQL first, then invalidates or refreshes Redis namespaces, all logical hot-key copies, and the in-process fallback copy. The guide uses this flow to explain invalidation and TTL tradeoffs.

**Blocked by:** 02: Request coalescing; 03: Cache warming; 04: Replicated hot key; 05: Local fallback cache.

**Status:** ready-for-agent

- [ ] The consistency read uses cache-aside behavior with an observable miss, hit, and configurable TTL.
- [ ] A successful update returns the changed book and a subsequent read through every cached lab route returns the new value.
- [ ] Every logical hot-key copy and the local fallback copy is invalidated or refreshed after the PostgreSQL commit.
- [ ] A failed PostgreSQL update does not report success or evict healthy cache entries.
- [ ] Unknown books return 404; invalid update input returns 400.
- [ ] HTTP-level acceptance checks cover cross-route read-after-update behavior and failed updates.
- [ ] The guide explains the remaining concurrent read/write race instead of claiming strict consistency.
