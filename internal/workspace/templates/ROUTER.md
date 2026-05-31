# ROUTER.md

## Session Root

Start every agent session at the workspace root — the directory containing this file.
Read the workspace docs here first, then navigate to the repo or worktree for code work.

Do not start a session inside a repo or worktree directly. Without the workspace root
as your starting point you are missing feature context, tool policy, routing rules,
and approval requirements.

**Context lives here. Code lives in repos. Commands run there.**

---

## Repos

These are the repositories this workspace orchestrates. Paths may be absolute or
relative to this workspace root. Run code commands with the repo path as cwd.

<!-- TODO: Run SETUP.md to populate this table. -->

| Name | Path | Purpose |
|------|------|---------|
| _example_ | _../my-app_ | _Application code, APIs, tests_ |

---

## Worktrees

For ticket work, never edit a repo directly. Use an isolated git worktree so
branches stay clean and multiple tickets can run in parallel.

Worktrees live inside this workspace under `worktrees/<repo-name>/<ticket-slug>`.
They are branched off the main repo, which may live anywhere on the filesystem.

To create a worktree for a ticket:
```
git -C <repo-path> worktree add <workspace>/worktrees/<repo-name>/<ticket-slug> -b <branch>
```

The worktree path for the active ticket is always recorded in `STATE.yaml` under
`next_action.cwd`. Use that path — do not guess or construct it manually.

---

## Feature State

Feature context lives in `features/<ticket-slug>/`. Read `STATE.yaml` for current
stage, owner, and worktree path. Read `TICKET.md`, `SPEC.md`, and `PLAN.md` for
background. Do not reconstruct state from memory — always read the files.

---

## Stages

Stage definitions live in `stages/<name>.md`. Flow control (order, worker per
stage, advance mode, repair loops) is declared in `workflows.yaml`.

| Stage          | Purpose                           |
|----------------|-----------------------------------|
| intake         | Load ticket context               |
| develop        | Feature implementation            |
| pr-open        | Open and submit a pull request    |
| pr-repair      | Fix CI failures or review feedback|
| qa-automation  | QA planning and test execution    |
