---
next_workflow: code-review
next_stage: code_review
advance: manual
model: claude-opus-4-7
effort: high
---

# Workflow: develop

> Before starting: read `workflows/REQUIREMENTS.md` for state update rules and error handling.

## Purpose

Implement the feature in a repo worktree and prepare it for a pull request.
Runs after the `intake` workflow has completed and `STATE.yaml` is `status: ready`.
Hands off to `pr-open` when implementation is done.

## Stages

```
implementation
```

### implementation

**Owner:** developer agent  
**Inputs:** `PLAN.md`, `SPEC.md`, repo worktree (see `STATE.yaml` → `next_action.cwd`)  
**Outputs:** `impl/QA_HANDOFF.md`, committed code on feature branch

Steps:
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
