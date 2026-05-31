---
next_workflow: pr-open
advance: auto
worker: zach-reviewer
---

# Workflow: code-review

> Before starting: read `workflows/REQUIREMENTS.md` for state update rules and error handling.

## Purpose

Review the implementation for correctness, spec compliance, and code quality before
a PR is opened. Automatically loops back to `develop` when changes are needed, up to
3 cycles. After 3 failed reviews a human is brought in.

## Cycle Counting

Before reviewing, count how many times `code-review` appears in `STATE.yaml` history
(entries where `workflow: code-review`). This is the current cycle number.

- Cycle 1–2: if changes needed, route back to `develop` automatically
- Cycle 3: if still not approved, call `orc wait` — a human must resolve

## Steps

**Owner:** zach-reviewer agent  
**Inputs:** feature branch code in worktree, `PLAN.md`, `SPEC.md`, `impl/QA_HANDOFF.md`,  
`impl/REVIEW.md` (from prior cycles)  
**Outputs:** `impl/REVIEW.md` (overwrite with updated findings)

1. Read `SPEC.md` and `PLAN.md` to understand the intended design.
2. Read `impl/QA_HANDOFF.md` for the implementation summary and known risks.
3. If a prior `impl/REVIEW.md` exists, read it to understand what was previously flagged.
4. Review the code changes in the worktree (`git diff main` or the feature branch).
5. Check for: correctness, spec compliance, edge cases, security issues, test coverage.
6. Write `impl/REVIEW.md` with findings using tags: `[bug]` `[spec]` `[style]` `[risk]` `[minor]`
7. Set verdict: `approved`, `needs-changes`, or `blocked`.

## Exit Criteria

`impl/REVIEW.md` is written with a clear verdict.

**If approved** — run:
```
orc advance <ticket> --workflow pr-open --owner <worker-id> --result "Code review passed"
```

**If needs-changes, cycle 1 or 2** — run:
```
orc advance <ticket> --workflow develop --owner <worker-id> --result "Review cycle <N>: changes needed — see impl/REVIEW.md"
```

**If needs-changes, cycle 3** — run:
```
orc wait <ticket> "3 review cycles failed — see impl/REVIEW.md for unresolved findings"
```

**If blocked** (design issue, spec conflict, missing requirement) — run:
```
orc wait <ticket> "Review blocked — <reason>. Human decision required before continuing."
```
