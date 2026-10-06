# 01: Runnable bookstore lab

**What to build:** A learner can start the Go/Gin API, PostgreSQL, and Redis together, read seeded books through a database-backed route, and call a distinct starter route for every cache exercise. The starter routes return the book through PostgreSQL with a visible `bypass` outcome until the learner completes their respective exercises. The consistency update route writes to PostgreSQL. Setup, reset, and basic request examples are documented.

**Blocked by:** None (can start immediately).

**Status:** done

- [x] One documented command starts the API, one PostgreSQL database, and one Redis instance; repeat setup does not duplicate books.
- [x] Seed books use the agreed book fields and include a designated popular book.
- [x] Health reports the API process; readiness reports PostgreSQL and Redis availability.
- [x] The baseline read returns a book from PostgreSQL, 404 for an unknown book, and 400 for an invalid ID.
- [x] All lab GET routes are reachable, return the same book shape, and identify the starter `bypass` behavior; the warming action clearly identifies its incomplete exercise.
- [x] The consistency update route persists a valid book update to PostgreSQL and returns the updated book.
- [x] A sample configuration and a short guide explain host and container startup, teardown, and the exercise TODOs.
- [x] Ordinary Go tests pass and verify the externally visible starter behavior through the HTTP API.
