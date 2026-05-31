# HOT-42 — Login 500 on cold start

**Type:** Hotfix  
**Priority:** P0  
**Reporter:** on-call  
**Reported:** 2026-05-31

## Summary

POST `/auth/login` returns 500 immediately after a cold deploy. Warm instances
are unaffected. Issue reproduces consistently on first request after startup.

## Stack Trace

```
NullPointerException: session store not initialized
  at SessionMiddleware.handle (middleware/session.go:42)
  at Router.ServeHTTP (router.go:88)
```

## Impact

All users unable to log in on fresh deploys. Rollback in place.

## Expected

Login succeeds within 200ms on first request after cold start.
