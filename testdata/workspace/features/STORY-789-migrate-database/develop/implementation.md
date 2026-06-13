# Develop — STORY-789

## Changes
- Added `users_v2` partitioned table and migration `0042_users_v2.sql`.
- Dual-write wrapper in `repo/users.go` gated by `MIGRATE_USERS_V2`.
- Backfill command `cmd/backfill-users`.

## Tests
- Unit tests for dual-write parity pass locally.
- Integration test for backfill idempotency added.
