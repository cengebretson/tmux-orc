# TICKET.md — STORY-789

## Summary

Migrate the `events` table from MySQL to PostgreSQL.

## Description

The `events` table is the last remaining table on the legacy MySQL instance.
All other tables were migrated to PostgreSQL last quarter. This ticket covers
writing the migration scripts, validating data integrity, and updating the
application to use the PostgreSQL connection for events queries.

The MySQL instance can be decommissioned once this is complete.

## Acceptance Criteria

- [ ] All rows from `events` (MySQL) are present in `events` (PostgreSQL) after migration
- [ ] Row counts and checksums match between source and destination
- [ ] Application reads and writes events from PostgreSQL in production
- [ ] MySQL `events` table is renamed to `events_migrated` (not dropped) as a safety net
- [ ] Rollback script tested and documented

## Links

- Ticket: https://stories.example.com/STORY-789
- PR: https://github.com/example/my-app/pull/91
- Infra issue: https://infra.example.com/issues/staging-env-down
