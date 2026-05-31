# AGENTS.md

## Scope

This is the workspace root. It owns shared routing, tool policy, feature context,
and cross-repo workflow state.

This workspace is designed to work equally well with Claude and Codex. `AGENTS.md`
is the shared source of truth. `CLAUDE.md` imports it; Codex reads it directly.
Never put product-specific instructions here — those belong in worker definitions.

## Read First

- Read `ROUTER.md` before deciding which repo or workflow owns the task.
- Read `TOOLS.md` before choosing commands, MCP servers, skills, scripts, or apps.
- Read `RULES.md` before writing files, opening PRs, or updating external systems.
- Read `workflows/REQUIREMENTS.md` before executing any workflow stage — it defines
  status values, STATE.yaml update rules, and error handling for all workflows.

## Feature Context

For ticket-driven work, first check `features/<ticket-slug>/`.
The feature folder is the durable source of ticket context across repos.

## Repo Commands

Run repo-specific commands with the selected repo or worktree as `cwd`.
Do not run package, test, or git commands from the workspace root unless
the workflow explicitly says to.

