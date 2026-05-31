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
    config/                       orc.yaml parsing (workspace config, repo list)
    health/                       workspace filesystem health checks
    state/                        STATE.yaml parsing and mutations
    workers/                      worker definition parsing and matching
    workflow/                     orc.yaml workflow section parsing (stage sequences, repair loops)
    stage/                        stage markdown file reading
    workspace/                    init, work, and template embedding
      templates/                  embedded workspace scaffold templates
        AGENTS.md, CLAUDE.md, ROUTER.md, TOOLS.md, RULES.md
        ORC.md                    agent state contract
        orc.yaml                  workspace config — repos, workflows, and settings
        features/_template/       feature context pack template
        workers/                  worker definition templates
          _template.md
          sample/                 sample workers (--with-sample-workers)
        stages/                   stage docs (plain markdown, no frontmatter)
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
| `orc status` | Show all features and their current workflow |
| `orc work <ticket>` | Create the feature folder for a ticket — run once by the human |
| `orc work <ticket> --tmux` | Also enable tmux session for this ticket |
| `orc show <ticket>` | Show full state for one ticket |
| `orc show <ticket> --json` | Full state as JSON for agent parsing |
| `orc next <ticket>` | Launch the next agent for a ticket |
| `orc next <ticket> --dry` | Preview the launch command without executing |
| `orc next <ticket> --json` | Next action as JSON for CI or scripting |
| `orc attach <ticket>` | Attach to the tmux session for a ticket |
| `orc start <ticket>` | Mark a ticket in_progress — called by the agent at the start of each session (hidden from help) |
| `orc advance <ticket> [--stage <stage>]` | Mark current stage complete and move to the next (called by agents, hidden from help) |
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
| Health check | `orc health` — filesystem validation, setup status, `orc.yaml` check (repos + workflows), `stages/` count |
| Feature lifecycle | `orc work`, `orc show`, `orc next`, `orc advance`, `orc wait`, `orc start`, `orc block`, `orc archive` |
| Status dashboard | `orc status` — active and archived features, table view |
| Worker routing | `orc.yaml` owns default worker per stage; overridden by `stage.owner` or `orc next --worker` |
| Multi-product | Claude and Codex launch commands rendered from worker `product` field |
| Workflows + stages | `orc.yaml` defines named pipelines with stage sequences, advance mode, and per-stage worker |
| Stage files | `stages/*.md` — plain markdown, no frontmatter; flow control lives entirely in `orc.yaml` |
| Repair loops | `repair_stages` section in `orc.yaml` with `repairs`, `worker`, `advance`, `max_retries` |
| Retry tracking | `stage_counts` map in STATE.yaml — incremented by `orc advance` |
| State mutations | `state.Advance`, `state.Block`, `state.WaitForHuman`, `state.Start`, `state.SetStatus` with history entries |
| Agent prompt scaffolding | Every `orc next` prompt includes preamble (read AGENTS.md + ORC.md, run `orc start`) and exact end-of-session command |
| JSON output | `orc show --json`, `orc next --json`, `orc status --json` — machine-readable for agent parsing and CI |
| Session contract | `ORC.md` at workspace root (replaces `REQUIREMENTS.md`); `AGENTS.md` Session Start section enforces state updates |
| Worktree cleanup | `orc archive` removes git worktrees, moves feature to `_archive/` |
| tmux integration | `orc work --tmux` opts in; `orc next` auto-creates session, sends agent to stage window; `orc attach` to jump in; runtime persisted in STATE.yaml |
| Tests | health, state, workers, workspace, workflow packages all covered |

### Planned

| Feature | Notes |
|---------|-------|
| ~~`orc tmux create/attach/list/kill`~~ | Done — `orc attach <ticket>`; sessions auto-created by `orc next`, one window per stage |
| `orc tui` | Bubble Tea dashboard — color-coded status, click to show/launch |
| ~~`orc run-next`~~ | Done — `orc next` now executes the agent directly; `--dry` to preview |
| ~~`--json` flag on `orc status`~~ | Done — `orc status --json` returns `{ active: [...], archived: [...] }` with full state objects |
| Banner suppression | Auto-suppress when stdout is not a TTY; `--no-banner` flag |
| `reasoning_effort` / `service_tier` in workers | Codex reasoning effort and priority tier in worker frontmatter, rendered in launch command |

### Future Enhancements

Ideas worth revisiting when the core is stable.

| Idea | Notes |
|------|-------|
| Resume prompt | When a session ends mid-stage without advancing, generate a recovery prompt summarizing what was done so far — reads existing output files, STATE.yaml history, and DECISIONS.md to reconstruct context for the next agent. Could live in `orc next` output or a dedicated `orc resume <ticket>` command. `next_action.prompt` in STATE.yaml is the natural place to write it. |
| Agent session completion notification | Notify the human when an agent finishes a stage — e.g. terminal bell, tmux alert, or a push notification via a configured webhook. Most useful in `--tmux` mode where the session runs unattended. |

### Deliberate divergences from original design

| Original | What we did instead | Why |
|----------|--------------------|----|
| `JIRA.md` in feature template | `TICKET.md` | System-agnostic — works with GitHub Issues, Linear, local files, or manual |
| `django/` subfolder in features | per-stage subfolders (`develop/`, `code-review/`, etc.) | Each stage writes to its own named folder — provenance is unambiguous |
| `orc workon` command | `orc work` | Shorter, cleaner |
| `orc done` command | `orc archive` with `_archive/` folder | Preserves history, keeps workspace clean, reversible |
| No first-run config | `SETUP.md` agent-driven setup | Cleaner than hand-editing files; works with Claude or Codex |
| Intake bundled into main workflow | Separate `intake` stage | Cleaner separation — every ticket goes through intake first, then routes to the right stage |
| `backend/` subfolder | per-stage subfolders | Stage name is the folder name — self-documenting and not coupled to any stack |

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
- Stage-assigned workers by default. Override with `--worker` for a single run or
  set `stage.owner` via `orc advance --owner` to persist across sessions.
- Product-agnostic by default. Every decision that could couple `orc` or the workspace
  to a single agent product should be reconsidered.
