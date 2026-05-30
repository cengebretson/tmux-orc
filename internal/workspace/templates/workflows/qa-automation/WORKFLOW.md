---
advance: manual
model: claude-sonnet-4-6
effort: medium
---

# Workflow: qa-automation

> Before starting: read `workflows/REQUIREMENTS.md` for state update rules and error handling.

## Purpose

Plan and implement automated tests for a completed feature, then collect evidence.
Runs after the PR has been reviewed and the `pr-open` workflow hands off here.

## Stages

```
qa_plan → qa_implementation → evidence
```

### qa_plan

**Owner:** documentor agent  
**Inputs:** `impl/QA_HANDOFF.md`, `TICKET.md`  
**Outputs:** `qa/SOURCE_CONTEXT.md`, `qa/QA_PLAN.md`

Steps:
1. Read `impl/QA_HANDOFF.md` and `TICKET.md`.
2. Write `qa/SOURCE_CONTEXT.md` with repo context the QA agent will need.
3. Draft `qa/QA_PLAN.md` with test cases, coverage goals, and tooling notes.

### qa_implementation

**Owner:** developer agent  
**Inputs:** `qa/QA_PLAN.md`, `qa/SOURCE_CONTEXT.md`, QA repo worktree  
**Outputs:** Tests committed, `qa/RUNS.md` updated

Steps:
1. Read `qa/QA_PLAN.md` and `qa/SOURCE_CONTEXT.md`.
2. Implement test cases in the QA repo worktree.
3. Run tests and record results in `qa/RUNS.md`.
4. Push and confirm CI passes.

### evidence

**Owner:** documentor agent  
**Inputs:** `qa/RUNS.md`, CI artifacts  
**Outputs:** `qa/QA_RESULT.md`, ticket updated in source system

Steps:
1. Read `qa/RUNS.md` and CI results.
2. Write `qa/QA_RESULT.md` with pass/fail summary and coverage notes.
3. Update the ticket in the source system (see `TOOLS.md` for the MCP server to use).
4. Run `orc archive <ticket>` to close out the feature.

## Exit Criteria

`qa/QA_RESULT.md` is complete with passing status, ticket is updated, and
`orc archive` has been run.
