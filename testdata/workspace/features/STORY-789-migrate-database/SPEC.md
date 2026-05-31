# SPEC.md — STORY-789

## Context

The events table on MySQL is the last blocker preventing decommission of the
legacy database instance. Keeping two databases running adds operational cost
and makes queries that join events with other tables unnecessarily complex
(requiring cross-database joins or data denormalization).

## Scope

### In scope
- `scripts/migrate_events.py` — bulk copy with integrity validation
- Schema translation: MySQL `TINYINT(1)` → PostgreSQL `BOOLEAN`, `DATETIME` → `TIMESTAMPTZ`
- Application query layer update: switch `EventRepository` to PostgreSQL connection
- Rollback procedure: rename PG table, point app back to MySQL

### Out of scope
- Streaming replication / zero-downtime cutover (scheduled maintenance window)
- Migration of other tables (already done)
- MySQL instance teardown (separate ops ticket)

## Behavior

**Migration script (`scripts/migrate_events.py`)**
1. Read rows from MySQL `events` in batches of 1000
2. Transform column types as documented
3. Insert into PostgreSQL `events` using `COPY` for performance
4. After all rows: compare row count and `SHA256(GROUP_CONCAT(id ORDER BY id))`
5. Exit non-zero if counts or checksums diverge

**Application cutover**
- Feature flag `USE_PG_EVENTS` gates which connection the `EventRepository` uses
- Flag is off in production until migration is validated in staging

## Open Questions

- [ ] Staging environment must be available to validate before production cutover.
      Currently blocked — see blocker in STATE.yaml.
