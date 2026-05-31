---
advance: manual
worker: fred-documentor
---

# Workflow: qa-automation

> Before starting: read `workflows/REQUIREMENTS.md` for state update rules and error handling.

## Purpose

Implement Playwright (or other) test automation from an existing QA plan.
Use this when the QA plan already exists and only implementation is needed.

## Stages

```
qa_implementation → evidence
```

### qa_implementation

**Owner:** bob-developer  
**Inputs:** `qa/QA_PLAN.md`, `qa/SOURCE_CONTEXT.md`, QA repo worktree  
**Outputs:** Tests committed, `qa/RUNS.md` updated

Steps:
1. Read `qa/QA_PLAN.md` and `qa/SOURCE_CONTEXT.md`.
2. Implement test cases in the QA repo worktree.
3. Run tests locally or against staging.
4. Record result in `qa/RUNS.md`.
5. Open a PR in the QA repo.
6. Update `STATE.yaml` to `evidence`.

### evidence

**Owner:** human or fred-documentor  
**Inputs:** `qa/RUNS.md`, CI artifacts  
**Outputs:** `qa/QA_RESULT.md`, Jira comment

## Exit Criteria

`qa/QA_RESULT.md` is complete and Jira is updated with evidence.
