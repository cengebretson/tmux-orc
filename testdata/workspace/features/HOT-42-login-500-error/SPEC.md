# SPEC.md — HOT-42

## Context

POST /auth/login returns 500 on cold start. Root cause: session store client
is initialized lazily and the first request arrives before initialization
completes. The null dereference is in `session/store.go`.

## Scope

### In scope
- Fix null dereference in session store initialization path
- Ensure store is ready before the HTTP server starts accepting connections
- Regression test that exercises login immediately after startup

### Out of scope
- Session store refactor or connection pooling changes
- Other endpoints — login path only

## Acceptance Criteria

- POST /auth/login returns 200 (or 401 for bad credentials) on first request after cold start
- No 500 in the cold-start window under load test
- Existing session tests still pass
