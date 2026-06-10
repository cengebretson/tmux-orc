# SPEC.md — PROJ-099

## Context

Users who forget their password have no self-service recovery path. Support
tickets for manual resets are the second-highest volume category. A standard
email-token reset flow will eliminate most of them.

## Scope

### In scope
- POST /auth/reset-request — accept email, send token if account exists
- POST /auth/reset-confirm — accept token + new password, invalidate token
- Token: 32-byte random hex, 1-hour TTL, single-use
- Email: plain-text only, contains reset link with token
- Rate limiting: 3 requests per email per hour

### Out of scope
- SMS / phone reset
- Admin-initiated resets (existing support flow stays)
- Password strength enforcement changes

## Behavior

**Request reset**
```
POST /auth/reset-request
{ "email": "user@example.com" }
→ 204 always (no account enumeration)
```

**Confirm reset**
```
POST /auth/reset-confirm
{ "token": "<hex>", "password": "<new>" }
→ 200 on success
→ 400 if token expired, used, or malformed
→ 422 if password fails validation
```

## Open Questions

- [ ] Token storage: users table column vs separate reset_tokens table? Prefer separate.
- [ ] Email sender: use existing mailer service or inline SMTP client?
