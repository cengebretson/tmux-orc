---
next_workflow: pr-open
advance: auto
worker: bob-developer
---

# Workflow: pr-repair

> Before starting: read `workflows/REQUIREMENTS.md` for state update rules and error handling.

## Purpose

Repair a PR that has CI failures, review feedback, or merge conflicts.

## Steps

**Owner:** developer agent  
**Inputs:** `impl/PR.md`, CI output, review comments  
**Outputs:** Fixed commits, updated `impl/PR.md`

1. Read `impl/PR.md` for PR URL and current status.
2. Identify the failure: CI, review feedback, or conflict.
3. Apply fixes in the app worktree.
4. Run local validation for the changed files.
5. Push fixes and check CI.
6. Update `impl/PR.md` with new status.

## Cost Note

Prefer bob-fast-fixer for lint, type, or small selector fixes. Use bob-developer
for test failures or logic issues. Escalate to ada-architect only for design-level problems.

## Exit Criteria

CI is green and all blocking review comments are resolved.

When done, run:
```
orc advance <ticket> --workflow pr-open --owner <worker-id> --result "PR repaired — CI green"
```

If the issue cannot be resolved:
```
orc block <ticket> "<what is blocking and what a human needs to decide>"
```
