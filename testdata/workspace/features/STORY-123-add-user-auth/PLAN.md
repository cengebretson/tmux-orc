# PLAN.md — STORY-123

## Approach

Add auth as a middleware layer in the gateway. Keep the implementation isolated
in `internal/middleware/auth.go` so it can be tested independently. Endpoints
live in `internal/handler/auth.go`. Use the existing secrets manager client to
load the RS256 key pair at startup.

## Steps

- [x] Load RS256 key pair from secrets manager on startup (`internal/secrets`)
- [x] Implement `ValidateToken(tokenString) (*Claims, error)` in middleware
- [x] Implement `POST /auth/login` handler
- [ ] Implement `POST /auth/refresh` handler
- [ ] Add `Protected()` route group wrapper
- [ ] Wire middleware into router
- [ ] Write unit tests: valid token, expired token, malformed token, missing header
- [ ] Write integration test for login + refresh flow

## Risk / Unknowns

- Key rotation: if the private key changes, all issued tokens become invalid.
  Need to confirm ops procedure before shipping.
- Refresh token revocation: stateless approach means we can't invalidate individual
  tokens without a blocklist. Deferred for now per product decision.

## QA Notes

- Test with tokens signed by a different key (should 401)
- Test clock skew: token expiring in the next second
- Verify that `user_id` and `email` are correctly propagated to downstream services
  via the `X-User-Id` and `X-User-Email` forwarded headers
