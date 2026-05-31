---
next_workflow: code-review
advance: manual
worker: bob-developer
---

# Workflow: develop

> Before starting: read `workflows/REQUIREMENTS.md` for state update rules and error handling.

## Purpose

Implement the feature in a repo worktree and prepare it for code review.
Runs after the `intake` workflow completes.

## Steps

**Owner:** developer agent  
**Inputs:** `PLAN.md`, `SPEC.md`, repo worktree (see `STATE.yaml` → `next_action.cwd`)  
**Outputs:** `impl/QA_HANDOFF.md`, committed code on feature branch

1. Read `SPEC.md` and `PLAN.md` for context.
2. Implement the feature in the repo worktree.
3. Write and run local tests for changed files.
4. Write `impl/QA_HANDOFF.md` with an implementation summary, test instructions, and known risks.
5. Commit all changes to the feature branch.

## Exit Criteria

Code is committed, `impl/QA_HANDOFF.md` is written, and local tests pass.

When done, run:
```
orc wait <ticket> "Implementation complete — human review before code review"
```
