# PLAN.md — PROJ-099

## Approach

Add a `reset_tokens` table to avoid touching the users schema. Keep token
generation and validation in a dedicated service layer so handlers stay thin.
Reuse the existing mailer for sending.

## Steps

- [x] Migration: create `reset_tokens(id, user_id, token_hash, expires_at, used_at)`
- [x] `internal/auth/reset.go` — `RequestReset`, `ConfirmReset`, `expireTokens` (cron)
- [ ] POST /auth/reset-request handler + rate limiter middleware
- [ ] POST /auth/reset-confirm handler
- [ ] Email template: `templates/email/password_reset.txt`
- [ ] Unit tests: token expiry, single-use enforcement, bad token rejection
- [ ] Integration test: full request → confirm flow

## Risk / Unknowns

- Token must be stored as a hash (not plaintext) — use SHA-256 of the raw bytes
- Rate limiter needs a Redis key scoped to email+IP to prevent enumeration via timing
