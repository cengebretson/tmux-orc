# RULES.md

## Permission and Approval Rules

Ask before doing any action that is destructive, external, costly,
security-sensitive, or hard to undo.

### Always ask first

- Deleting files, branches, worktrees, or feature context folders.
- Rewriting Git history: rebase, force-push, reset, or amend published commits.
- Installing or upgrading dependencies.
- Running commands that modify production, staging, Jira, GitHub PRs, or CI state.
- Starting long-running background agents or services.
- Running broad test suites or commands expected to take more than 10 minutes.
- Reading or changing secrets, credentials, env files, auth tokens, or private config.
- Sending notifications or comments to external systems.
- Changing shared workspace rules, templates, or setup scripts.

### Usually okay without asking

- Reading files.
- Searching the repo.
- Creating or editing files inside the current ticket context.
- Running targeted local validation for files just changed.
- Creating draft plans, summaries, or proposed commands.
- Updating `STATE.yaml` to reflect work just completed, unless the update changes
  ownership, closes a ticket, or marks external status.

### Ask if unclear

If an action could surprise the human, ask before doing it.
If an action affects another person, another system, or shared state, ask before doing it.

---

## State Update Rule

Every agent or script that performs work for a feature must keep
`features/<ticket-slug>/STATE.yaml` current.

Update `STATE.yaml` whenever any of these change:

- `status`
- `stage.name`
- `stage.worker`
- `next_action`
- required or completed outputs
- active repo/worktree paths
- tmux session or window names
- human attention requirements
- blocker state
- completion state

Before ending an agent session, update `STATE.yaml` so `orc status`,
`orc next`, and `orc stuck` reflect reality.

---

## Worktree Location

All agent-created Git worktrees should live under the workspace-level `worktrees/` folder.

Shape:

```
worktrees/
  <repo-name>/
    <ticket-slug>/
```

Do not create ad hoc worktrees inside product repos unless a repo-specific rule explicitly
requires it. Record the active worktree path in `features/<ticket-slug>/STATE.yaml`.

---

## Stage Quality

The stage files in `stages/` are the intelligence of the system — they define
what agents do, when they're done, and how they hand off to the next step.

A stage doc should have:
- **Clear exit criteria** — the agent knows exactly when it's done
- **Explicit output definitions** — which files get written and where
- **Unambiguous signals** — structured fields like `verdict:` that downstream
  stages can read without interpretation
- **Exact commands** — `orc mark ... advance`, `orc mark ... wait`, or `orc mark ... block` for every outcome,
  with no ambiguity about which to use

The sample workflows are a starting point. Tune them to your stack, review
standards, and team process. They are plain markdown files — edit one and the
next agent session picks up the new instructions immediately. No deploys, no
code changes required.

---

## Cost-Aware Worker Selection

Prefer the lowest-cost worker that is allowed for the workflow/stage and capable of the task.
Escalate model, thinking level, or service tier only when the state, workflow, or human says
the complexity requires it.

Ask before using high-cost workers for routine implementation, lint fixes, or small test fixes.
Record cost-tier escalation in `STATE.yaml` history.
