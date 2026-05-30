# Workflow: pr-open

> Before starting: read `workflows/REQUIREMENTS.md` for state update rules and error handling.

## Purpose

Open a pull request after implementation is complete. Runs final validation,
writes the PR description, opens the PR, and hands off for review.

## Stages

```
pr_preflight → pr_create → pr_handoff
```

### pr_preflight

**Owner:** developer agent  
**Inputs:** `impl/QA_HANDOFF.md`, feature worktree  
**Outputs:** Clean branch, passing local checks

Steps:
1. Read `features/<ticket-slug>/STATE.yaml` for the active worktree and branch.
2. Run local validation: tests, lint, type checks for changed files only.
3. If checks fail, set `STATE.yaml` `status: blocked` with failure details in
   `next_action.prompt` and stop — do not open a PR against a broken branch.
4. Ensure branch is rebased or merged against the base branch with no conflicts.
5. Push the branch to the remote.
6. Advance `STATE.yaml` to `pr_create`.

### pr_create

**Owner:** developer agent  
**Inputs:** `impl/QA_HANDOFF.md`, `TICKET.md`, commit log  
**Outputs:** Open PR, populated `impl/PR.md`

Steps:
1. Read `TICKET.md` for the ticket summary and acceptance criteria.
2. Read `impl/QA_HANDOFF.md` for the implementation summary.
3. Write a PR title: concise, under 70 characters, describes what changed.
4. Write a PR body covering:
   - What changed and why (link to ticket)
   - How to test it
   - Any migration steps or deployment notes
5. Open the PR via the source control MCP server (see `TOOLS.md`).
6. Write the PR URL and status to `impl/PR.md`.
7. Advance `STATE.yaml` to `pr_handoff`.

### pr_handoff

**Owner:** human  
**Inputs:** `impl/PR.md`  
**Outputs:** PR approved or feedback logged

Steps:
1. Review the open PR.
2. If changes are requested, use the `pr-repair` workflow.
3. When approved and merged, run `orc archive <ticket>` to close out the feature.

## Exit Criteria

PR is open, `impl/PR.md` has the URL, and `STATE.yaml` points to `pr_handoff`
with `owner: human`.

## Error Handling

If preflight checks fail:
- Set `status: blocked` in `STATE.yaml`
- Set `next_action.prompt` to the specific failure and what needs fixing
- Do not push or open a PR
