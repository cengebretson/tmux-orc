# PROJ-099 — Password Reset Flow

## Summary

Users cannot reset their password via email. The forgot-password link returns a 500.

## Acceptance criteria

- User receives a reset email within 60 seconds of requesting one
- Reset token expires after 1 hour
- Token is single-use — invalidated after first click
- Redirect to login on success with a confirmation message

## Notes

Auth service owns token generation. Email service is a separate microservice — use the existing `email.Send` interface.
