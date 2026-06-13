# PR — STORY-789

**PR:** my-app#812 — "Migrate users table to partitioned v2 schema"

## Summary
Dual-write + backfill behind `MIGRATE_USERS_V2`. No read-path change until backfill verifies.

## CI
- ✅ unit
- ✅ lint
- ❌ staging-deploy — staging environment unavailable
