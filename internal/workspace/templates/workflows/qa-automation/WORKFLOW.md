---
advance: manual
worker: fred-documentor
---

# Workflow: qa-automation

> Before starting: read `workflows/REQUIREMENTS.md` for state update rules and error handling.

## Purpose

Plan and implement automated tests for a completed feature, then collect evidence.
Runs after the PR has been reviewed and the `pr-open` workflow hands off here.

## Steps

**Owner:** documentor agent  
**Inputs:** `impl/QA_HANDOFF.md`, `TICKET.md`, QA repo worktree  
**Outputs:** `qa/SOURCE_CONTEXT.md`, `qa/QA_PLAN.md`, `qa/RUNS.md`, `qa/QA_RESULT.md`

**QA Planning**
1. Read `impl/QA_HANDOFF.md` and `TICKET.md`.
2. Write `qa/SOURCE_CONTEXT.md` with repo context the QA agent will need.
3. Draft `qa/QA_PLAN.md` with test cases, coverage goals, and tooling notes.

**Implementation**
4. Read `qa/QA_PLAN.md` and `qa/SOURCE_CONTEXT.md`.
5. Implement test cases in the QA repo worktree.
6. Run tests and record results in `qa/RUNS.md`.
7. Push and confirm CI passes.

**Evidence**
8. Read `qa/RUNS.md` and CI results.
9. Write `qa/QA_RESULT.md` with pass/fail summary and coverage notes.
10. Update the ticket in the source system (see `TOOLS.md` for the MCP server to use).
11. Run `orc archive <ticket>` to close out the feature.

## Exit Criteria

`qa/QA_RESULT.md` is complete with passing status, ticket is updated, and
`orc archive` has been run.
