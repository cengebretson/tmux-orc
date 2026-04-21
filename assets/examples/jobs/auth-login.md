---
pipeline: frontend
domain: src/frontend/auth/login/
---

## Goal
Build the login flow with JWT token handling.

## Acceptance criteria
- Email + password form with client-side validation
- JWT stored in httpOnly cookie, not localStorage
- Redirects to /dashboard on success
- Shows inline field-level errors on failure
- Mobile responsive

## Context
Designs are in Figma at [link]. Backend JWT endpoint already exists at `POST /api/auth/login`
and returns `{ token, user }`. The existing `useAuth` hook at `src/shared/hooks/useAuth.ts`
should be extended, not replaced.

## Related
- Linear: AUTH-42
- Backend PR: #118 (merged, branch agent/auth-api)
