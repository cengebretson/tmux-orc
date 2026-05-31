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

Repos and their paths are declared in `orc.yaml`. Read that file to find repo names,
filesystem paths, and purposes. Use the path from `orc.yaml` as `cwd` when running
code commands — do not guess paths.

For ticket work, use the worktree path from `STATE.yaml` → `next_action.cwd`, not
the repo path directly.

---

## Routing

When deciding which repo owns a task, start with the ticket's feature folder:

1. Read `features/<ticket-slug>/STATE.yaml` — `next_action.cwd` is the worktree to use.
2. If there is no active ticket, check `orc.yaml` for repo purposes and pick the one
   that matches the work type.
3. Run code commands (tests, lint, build) from the worktree or repo path, not the
   workspace root.

---

## Worktrees

For ticket work, never edit a repo directly. Use an isolated git worktree so
branches stay clean and multiple tickets can run in parallel.

Worktrees live inside this workspace under `worktrees/<repo-name>/<ticket-slug>`.

To create a worktree for a ticket:
```
git -C <repo-path> worktree add <workspace>/worktrees/<repo-name>/<ticket-slug> -b <branch>
```

The worktree path for the active ticket is always recorded in `STATE.yaml` under
`next_action.cwd`. Use that path — do not guess or construct it manually.

---

## Stages

Stage definitions live in `stages/<name>.md`. Flow control (order, worker per
stage, advance mode, repair loops) is declared in `orc.yaml`.
