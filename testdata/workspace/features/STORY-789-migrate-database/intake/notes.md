# Intake Notes — STORY-789

Migrate the legacy `users` table to the new partitioned schema without downtime.

## Scope
- Dual-write behind a feature flag during cutover.
- Backfill job for historical rows.
- Read path switches to the new table once backfill verifies.

## Out of scope
- Deprecating the old table (follow-up ticket).
