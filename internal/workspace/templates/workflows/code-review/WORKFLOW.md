---
next_workflow: pr-open
next_stage: pr_preflight
advance: auto
worker: zach-reviewer
---

# Workflow: code-review

> Before starting: read `workflows/REQUIREMENTS.md` for state update rules and error handling.

## Purpose

Review the implementation for correctness, spec compliance, and code quality before
a PR is opened. Runs after `develop` completes and hands off to `pr-open` when the
review passes.

## Stages

```
code_review
```

### code_review

**Owner:** zach-reviewer agent  
**Inputs:** feature branch code in worktree, `PLAN.md`, `SPEC.md`, `impl/QA_HANDOFF.md`  
**Outputs:** `impl/REVIEW.md`

Steps:
1. Read `SPEC.md` and `PLAN.md` to understand the intended design.
2. Read `impl/QA_HANDOFF.md` for the implementation summary and known risks.
3. Review the code changes in the worktree (`git diff main` or the feature branch).
4. Check for: correctness, spec compliance, edge cases, security issues, test coverage.
5. Write `impl/REVIEW.md` with findings — approved, needs changes, or blocked.

## Exit Criteria

`impl/REVIEW.md` is written with a clear verdict.

**If approved** — run:
```
orc advance <ticket> pr_preflight --workflow pr-open --owner <worker-id> --result "Code review passed"
```

**If changes needed** — run:
```
orc wait <ticket> "Review found issues — see impl/REVIEW.md before opening PR"
```
