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

## Session Start

At the start of every ticket session, before doing any work:

1. Identify the ticket from your prompt or context
2. Run `orc start <ticket>` to mark the ticket in_progress
3. Run `orc show <ticket> --json` to read current state — note `stage.workflow`
4. Read `features/<ticket-slug>/STATE.yaml` for full feature context
5. Read `workflows/<workflow>/WORKFLOW.md` for the current workflow instructions

At the end of every session, run exactly one of:
- `orc advance <ticket> --workflow <next-workflow> --owner <who> --result "<what was done>"` — workflow complete
- `orc wait <ticket> "<what you need from the human>"` — need human input or approval
- `orc block <ticket> "<what is preventing progress>"` — external blocker

Never end a session without updating state. Never hand-edit STATE.yaml directly.

## Feature Context

For ticket-driven work, the feature folder is the durable source of truth.
Everything the agent needs to pick up and continue is in `features/<ticket-slug>/`.

## Repo Commands

Run repo-specific commands with the selected repo or worktree as `cwd`.
Do not run package, test, or git commands from the workspace root unless
the workflow explicitly says to.

