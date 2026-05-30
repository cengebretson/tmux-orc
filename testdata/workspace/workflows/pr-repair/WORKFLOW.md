# Workflow: pr-repair

> Before starting: read `workflows/REQUIREMENTS.md` for state update rules and error handling.

## Purpose

Repair a PR that has CI failures, review feedback, or merge conflicts.

## Stages

```
pr_repair → (back to previous workflow stage)
```

### pr_repair

**Owner:** bob-developer (or bob-fast-fixer for small fixes)  
**Inputs:** `impl/PR.md`, CI output, review comments  
**Outputs:** Fixed commits, updated `impl/PR.md`

Steps:
1. Read `impl/PR.md` for PR URL and current status.
2. Identify the failure: CI, review feedback, or conflict.
3. Apply fixes in the app worktree.
4. Run local validation for the changed files.
5. Push fixes and check CI.
6. Update `impl/PR.md` with new status.
7. Update `STATE.yaml` — if repaired, advance back to the previous stage; if blocked, set `status: blocked`.

## Cost Note

Prefer bob-fast-fixer for lint, type, or small selector fixes. Use bob-developer
for test failures or logic issues. Escalate to ada-architect only for design-level problems.

## Exit Criteria

CI is green and all blocking review comments are resolved.
