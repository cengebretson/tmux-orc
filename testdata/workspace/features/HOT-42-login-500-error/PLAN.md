# PLAN.md — HOT-42

## Approach

Block server startup until the session store is healthy. Use a readiness check
in main before binding the listener — this is the safest fix with no risk to
the session contract.

## Steps

- [ ] Add `store.WaitReady(ctx)` that polls the store connection with a timeout
- [ ] Call `WaitReady` in `main.go` before `http.ListenAndServe`
- [ ] Add regression test: create server, hit login before any warmup delay
- [ ] Verify existing session tests pass unchanged

## Risk

- `WaitReady` must have a timeout — if the store is down permanently, startup
  should fail fast rather than hang. Use 10s default.
