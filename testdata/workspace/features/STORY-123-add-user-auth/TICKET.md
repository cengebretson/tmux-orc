# TICKET.md — STORY-123

## Summary

Add JWT-based user authentication to the API gateway.

## Description

The API currently has no authentication layer. All endpoints are open. We need to
add JWT authentication so that protected routes require a valid token. The auth
service issues tokens on login; the gateway validates them on every request.

Refresh tokens should be supported with a 7-day TTL. Access tokens expire after
15 minutes.

## Acceptance Criteria

- [ ] `POST /auth/login` returns `{ access_token, refresh_token }`
- [ ] Protected routes return 401 if no token or token is expired
- [ ] `POST /auth/refresh` exchanges a valid refresh token for a new access token
- [ ] Tokens are signed with RS256 using the key pair in secrets manager
- [ ] Auth middleware is tested with valid, expired, and malformed tokens

## Links

- Ticket: https://stories.example.com/STORY-123
- PR: (pending)
