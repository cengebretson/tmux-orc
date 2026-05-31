# CLAUDE.md

## Project Overview

**orc** is a Go CLI for agentic workspace orchestration. It scaffolds and manages a
filesystem-based workspace where agents (Claude, Codex, Cursor, etc.) carry feature
work across repos — from ticket intake through implementation, PR repair, QA automation,
and evidence collection.

The core idea: durable state lives in files (`STATE.yaml`, markdown docs), not in memory.
Policy lives in files (`RULES.md`, `AGENTS.md`, worker definitions), not in code.
`orc` reads, validates, renders, and updates — it does not encode workflow logic.

## Repository Layout

```
orc/
  cmd/orc/main.go                 CLI entry point (Cobra)
  internal/
    health/                       workspace filesystem health checks
    state/                        STATE.yaml parsing and mutations
    workers/                      worker definition parsing and matching
    workflow/                     WORKFLOW.md frontmatter parsing
    workspace/                    init, work, and template embedding
      templates/                  embedded workspace scaffold templates
        AGENTS.md, CLAUDE.md, ROUTER.md, TOOLS.md, RULES.md
        features/_template/       feature context pack template
        workers/                  worker definition templates
          _template.md
          sample/                 sample workers (--with-sample-workers)
        workflows/                workflow docs, REQUIREMENTS.md, commands.yaml
  docs/                           design documentation (HTML)
  assets/                         roles and skills reference files
  go.mod
```

## Build and Run

```bash
go build -o orc ./cmd/orc/...
./orc --help
./orc init --dry-run
./orc init --workspace ~/Desktop/my-workspace --with-sample-workers
```

## Tests

```bash
go test ./...
```

## Commands

| Command | Description |
|---------|-------------|
| `orc init` | Scaffold a new workspace |
| `orc init --workspace <path>` | Scaffold at a specific path |
| `orc init --with-sample-workers` | Include sample worker files |
| `orc init --dry-run` | Preview without writing |
| `orc init --force` | Overwrite existing files |
| `orc health` | Check workspace filesystem health |
| `orc status` | Show all features and their current stage |
| `orc work <ticket>` | Create the feature folder for a ticket — run once by the human |
| `orc show <ticket>` | Show full state for one ticket |
| `orc show <ticket> --json` | Full state as JSON for agent parsing |
| `orc next <ticket>` | Launch the next agent for a ticket |
| `orc next <ticket> --dry` | Preview the launch command without executing |
| `orc next <ticket> --json` | Next action as JSON for CI or scripting |
| `orc start <ticket>` | Mark a ticket in_progress — called by the agent at the start of each session (hidden from help) |
| `orc advance <ticket> [--workflow <wf>]` | Mark current workflow complete and move to the next (called by agents, hidden from help) |
| `orc wait <ticket> <reason>` | Mark a ticket as waiting for human input |
| `orc block <ticket> <reason>` | Mark a ticket as blocked |
| `orc archive <ticket>` | Archive a completed feature, remove worktrees |

## Roadmap

The original design document lives in `docs/`. This section tracks what's been
built, what's planned, and where we deliberately diverged from the original plan.

### Implemented

| Area | What exists |
|------|-------------|
| Workspace scaffold | `orc init` with embedded templates, `--with-sample-workers`, `--dry-run`, `--force` |
| Configuration | `SETUP.md` — agent-driven setup (not in original design, added improvement) |
| Health check | `orc health` — filesystem validation, setup status, required workflow check, frontmatter validation |
| Feature lifecycle | `orc work`, `orc show`, `orc next`, `orc advance`, `orc wait`, `orc start`, `orc block`, `orc archive` |
| Status dashboard | `orc status` — active and archived features, table view |
| Worker routing | Workflow owns default worker via `worker:` in WORKFLOW.md frontmatter; overridden by `stage.owner` or `orc next --worker` |
| Multi-product | Claude and Codex launch commands rendered from worker `product` field |
| Workflows | intake, develop, code-review, pr-open, pr-repair, qa-automation — each with WORKFLOW.md frontmatter |
| Workflow frontmatter | `next_workflow`, `advance` (auto/manual), `worker` (default worker ID) |
| Cross-workflow transitions | `orc advance --workflow` updates `stage.workflow`; worker resolved from new workflow's frontmatter |
| State mutations | `state.Advance`, `state.Block`, `state.WaitForHuman`, `state.Start`, `state.SetStatus` with history entries |
| Agent prompt scaffolding | Every `orc next` prompt includes preamble (read AGENTS.md, run `orc start`) and exact end-of-session command |
| JSON output | `orc show --json`, `orc next --json`, `orc status --json` — machine-readable for agent parsing and CI |
| Session contract | `REQUIREMENTS.md` shared workflow contract; `AGENTS.md` Session Start section enforces state updates |
| Worktree cleanup | `orc archive` removes git worktrees, moves feature to `_archive/` |
| Tests | health, state, workers, workspace packages all covered |

### Planned

| Feature | Notes |
|---------|-------|
| ~~`orc tmux create/attach/list/kill`~~ | Done — `orc attach <ticket>`; sessions auto-created by `orc next`, one window per workflow |
| `orc tui` | Bubble Tea dashboard — color-coded status, click to show/launch |
| ~~`orc run-next`~~ | Done — `orc next` now executes the agent directly; `--dry` to preview |
| ~~`--json` flag on `orc status`~~ | Done — `orc status --json` returns `{ active: [...], archived: [...] }` with full state objects |
| Banner suppression | Auto-suppress when stdout is not a TTY; `--no-banner` flag |
| `reasoning_effort` / `service_tier` in workers | Codex reasoning effort and priority tier in worker frontmatter, rendered in launch command |

### Deliberate divergences from original design

| Original | What we did instead | Why |
|----------|--------------------|----|
| `JIRA.md` in feature template | `TICKET.md` | System-agnostic — works with GitHub Issues, Linear, local files, or manual |
| `django/` subfolder in features | `impl/` | Not framework-specific |
| `orc workon` command | `orc work` | Shorter, cleaner |
| `orc done` command | `orc archive` with `_archive/` folder | Preserves history, keeps workspace clean, reversible |
| No first-run config | `SETUP.md` agent-driven setup | Cleaner than hand-editing files; works with Claude or Codex |
| Intake bundled into main workflow | Separate `intake` workflow | Cleaner separation — every ticket goes through intake first, then routes to the right workflow |
| `backend/` subfolder | `impl/` subfolder | Generic — not coupled to backend/frontend distinction |

## Template System

Templates are embedded in the binary via `//go:embed all:templates`.
The `all:` prefix is required to include directories starting with `_` (like `_template`).

To add a new template file, drop it under `internal/workspace/templates/` and rebuild.

## Hard Requirements

**orc and the workspaces it generates must work equally well for Claude and Codex.**

This is a non-negotiable design constraint. Concretely:

- The workspace scaffold (`AGENTS.md`, `CLAUDE.md`, worker files, workflow docs) must
  be readable and actionable by both Claude Code and Codex without modification.
- `CLAUDE.md` imports `AGENTS.md` as the shared source of truth. Codex reads `AGENTS.md`
  directly. The two must never diverge.
- `orc` CLI output (launch commands, prompts, next-action text) must be correct for
  whichever product the worker definition specifies — never assume Claude.
- Worker definitions use `product: claude` or `product: codex` (or others) in frontmatter.
  `orc next` renders the correct launch command for the active worker's product.
- Do not add features, flags, or template content that only makes sense for one product.
  If a feature is product-specific, gate it behind the worker's `product` field at runtime.

## Design Principles

- Policy in files, not code. Worker behavior, model choice, and cost tier live in
  markdown files. `orc` parses, matches, renders, and updates state.
- Durable state. `STATE.yaml` survives restarts, session changes, and agent switches.
- Human-in-the-loop first. Background execution comes last, after logging and recovery
  are solid.
- Workflow-assigned workers by default. Override with `--worker` for a single run or
  set `stage.owner` via `orc advance --owner` to persist across sessions.
- Product-agnostic by default. Every decision that could couple `orc` or the workspace
  to a single agent product should be reconsidered.
