# Workflow: develop

> Before starting: read `workflows/REQUIREMENTS.md` for state update rules and error handling.

## Purpose

Carry a ticket from implementation through QA automation and evidence collection.
Runs after the `intake` workflow has completed and STATE.yaml is `status: ready`.

## Stages

```
implementation → pr_open* → qa_plan → qa_implementation → evidence
```

(*) `pr_open` triggers the `pr-open` workflow. After the PR is merged, advance to `qa_plan`.

### implementation

**Owner:** developer agent  
**Inputs:** `PLAN.md`, `SPEC.md`, repo worktree  
**Outputs:** `impl/PR.md`, `impl/QA_HANDOFF.md`

Steps:
1. Read `SPEC.md` and `PLAN.md` for context.
2. Implement the feature in the repo worktree from `STATE.yaml`.
3. Write and run local tests for changed files.
4. Write `impl/QA_HANDOFF.md` with implementation summary, test instructions, and risks.
5. Trigger the `pr-open` workflow to open the PR.
6. Once the PR is open, record the URL in `impl/PR.md`.
7. Update `STATE.yaml` to `qa_plan`.

### qa_plan

**Owner:** documentor agent  
**Inputs:** `impl/QA_HANDOFF.md`, `TICKET.md`  
**Outputs:** `qa/SOURCE_CONTEXT.md`, `qa/QA_PLAN.md`

Steps:
1. Read `impl/QA_HANDOFF.md` and `TICKET.md`.
2. Populate `qa/SOURCE_CONTEXT.md` with repo context for the QA agent.
3. Draft `qa/QA_PLAN.md` with test cases and coverage plan.
4. Update `STATE.yaml` to `qa_implementation`.

### qa_implementation

**Owner:** developer agent  
**Inputs:** `qa/QA_PLAN.md`, QA repo worktree  
**Outputs:** Committed tests, `qa/RUNS.md` updated

Steps:
1. Read `qa/QA_PLAN.md` and `qa/SOURCE_CONTEXT.md`.
2. Implement tests in the QA repo worktree.
3. Run the test suite and record results in `qa/RUNS.md`.
4. Push and confirm CI passes.
5. Update `STATE.yaml` to `evidence`.

### evidence

**Owner:** documentor agent or human  
**Inputs:** `qa/RUNS.md`, CI artifacts  
**Outputs:** `qa/QA_RESULT.md`, ticket updated in source system

Steps:
1. Read `qa/RUNS.md` and CI results.
2. Write `qa/QA_RESULT.md` with pass/fail summary and coverage notes.
3. Update the ticket in the source system (see `TOOLS.md` for system and MCP server).
4. Run `orc archive <ticket>` to close out the feature.

## Exit Criteria

`qa/QA_RESULT.md` is complete with passing status, PR is merged, and the ticket
is updated in the source system.
