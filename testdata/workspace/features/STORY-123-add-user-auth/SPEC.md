# SPEC.md — STORY-123

## Context

The API gateway forwards all traffic to upstream services but does no auth itself.
Adding a middleware layer here lets all services trust that requests reaching them
are already validated, without each service re-implementing token parsing.

## Scope

### In scope
- JWT validation middleware (`internal/middleware/auth.go`)
- Login endpoint (`POST /auth/login`)
- Refresh endpoint (`POST /auth/refresh`)
- Route group helpers: `Protected()` and `Public()`
- Unit tests for middleware and both endpoints

### Out of scope
- User management (create, update, delete users)
- OAuth / third-party SSO
- Role-based access control (separate ticket)

## Behavior

**Login flow**
1. Client posts `{ email, password }` to `/auth/login`
2. Gateway calls the user service to validate credentials
3. On success: issue access token (15 min TTL) and refresh token (7 day TTL)
4. Tokens are RS256-signed using the key pair in secrets manager under `jwt/private`

**Validation middleware**
- Extracts `Authorization: Bearer <token>` header
- Validates signature, expiry, and issuer claim
- Attaches `user_id` and `email` to request context on success
- Returns `401 Unauthorized` with `{ error: "token_expired" }` or `{ error: "invalid_token" }` on failure

**Refresh flow**
1. Client posts `{ refresh_token }` to `/auth/refresh`
2. Gateway validates the refresh token (not expired, not revoked)
3. Issues a new access token; refresh token is not rotated

## Open Questions

- [x] Refresh token TTL — resolved: 7 days (confirmed 2026-05-30)
- [ ] Should refresh tokens be stored server-side for revocation, or stateless?
