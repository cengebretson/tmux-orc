# PLAN.md — STORY-789

## Approach

Run migration during a scheduled maintenance window. Use a Python script for
the bulk copy (easier to inspect and debug than a Go binary). Validate with
checksums before cutting over the application. Keep the feature flag off in
production until staging validation passes.

## Steps

- [x] Write `scripts/migrate_events.py` with batch copy and checksum validation
- [x] Write rollback script `scripts/rollback_events.sh`
- [x] Update `EventRepository` to support `USE_PG_EVENTS` feature flag
- [x] Add PostgreSQL schema migration: `migrations/0041_create_events.sql`
- [x] Test migration script against staging MySQL → staging PostgreSQL
- [ ] **BLOCKED** — staging environment unavailable, cannot validate
- [ ] Run migration in production maintenance window
- [ ] Verify production checksums
- [ ] Enable `USE_PG_EVENTS` flag in production
- [ ] Monitor for 24h, then rename MySQL table to `events_migrated`

## Risk / Unknowns

- Data volume: ~18M rows, estimated 25 min copy time at 1000 rows/batch.
  Can increase batch size to 5000 if time is a concern.
- The `metadata` column contains JSON stored as `TEXT` in MySQL.
  PostgreSQL schema uses `JSONB` — validated that all values parse correctly.

## QA Notes

- Run checksum comparison script after migration completes
- Verify application events (read + write) via smoke tests before re-opening to users
- Check that event timestamps are preserved correctly (UTC in both systems)
