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
    config/                       orc.yaml parsing — repos, workflows, repair stages, settings
    health/                       workspace filesystem health checks
    runner/                       next-action resolution — worker, prompt, launch args
    state/                        STATE.yaml parsing and mutations
    workers/                      worker definition parsing
    stage/                        stage markdown file reading
    resume/                       recovery prompt builder
    validate/                     per-ticket state validation
    tmux/                         tmux session management
    tui/                          Bubble Tea dashboard
    workspace/                    init, work, and template embedding
      templates/                  embedded workspace scaffold templates
  scripts/
    pre-commit                    tidy → fmt → lint → test (symlink to .git/hooks/pre-commit)
  go.mod
  Makefile
  plan.md                         active roadmap and future ideas
```

## Dev Workflow

```bash
make build   # go build -o orc ./cmd/orc/...
make test    # go test ./...
make lint    # golangci-lint (errcheck, govet, staticcheck, unused, ineffassign)
make check   # lint + test together
make fmt     # gofmt -w
make tidy    # go mod tidy
```

Install the pre-commit hook once after cloning:

```bash
ln -sf ../../scripts/pre-commit .git/hooks/pre-commit
```

The hook runs tidy → fmt → lint → test on every commit.

## Commands

See `README.md` for the full command reference split into human and agent commands.
Quick reference for dev/test use:

```bash
./orc init --dry-run
./orc init --workspace /tmp/test-ws --with-sample-workers
./orc health --workspace /tmp/test-ws
./orc work STORY-123 --workspace /tmp/test-ws
./orc next STORY-123 --dry --workspace /tmp/test-ws
./orc next STORY-123 --json --workspace /tmp/test-ws
```

## Template System

Templates are embedded in the binary via `//go:embed all:templates`.
The `all:` prefix is required to include directories starting with `_` (like `_template`).

To add a new template file, drop it under `internal/workspace/templates/` and rebuild.

## Hard Requirements

**orc and the workspaces it generates must work equally well for Claude and Codex.**

- The workspace scaffold must be readable and actionable by both without modification.
- `CLAUDE.md` imports `AGENTS.md` as the shared source of truth. Codex reads `AGENTS.md`
  directly. The two must never diverge.
- `orc` CLI output must be correct for whichever product the worker specifies — never
  assume Claude.
- Do not add features or template content that only makes sense for one product.
  Gate product-specific behavior behind the worker's `product` field at runtime.

## Design Principles

- **Policy in files, not code.** Worker behavior, model choice, and cost tier live in
  markdown files. `orc` parses, matches, renders, and updates state.
- **Durable state.** `STATE.yaml` survives restarts, session changes, and agent switches.
- **Human-in-the-loop first.** Background execution comes last, after logging and recovery
  are solid.
- **Stage-assigned workers by default.** Override with `--worker` for a single run or
  set `stage.owner` via `orc advance --owner` to persist across sessions.
- **Product-agnostic by default.** Every decision that could couple `orc` to a single
  agent product should be reconsidered.

## Deliberate Divergences from Original Design

| Original | What we did instead | Why |
|----------|--------------------|----|
| `JIRA.md` in feature template | `TICKET.md` | System-agnostic — works with GitHub Issues, Linear, local files, or manual |
| `django/` subfolder in features | per-stage subfolders (`develop/`, `code-review/`, etc.) | Each stage writes to its own named folder — provenance is unambiguous |
| `orc workon` command | `orc work` | Shorter, cleaner |
| `orc done` command | `orc archive` with `_archive/` folder | Preserves history, keeps workspace clean, reversible |
| No first-run config | `SETUP.md` agent-driven setup | Cleaner than hand-editing files; works with Claude or Codex |
| Intake bundled into main workflow | Separate `intake` stage | Cleaner separation — every ticket goes through intake first |
| `backend/` subfolder | per-stage subfolders | Stage name is the folder name — self-documenting and not coupled to any stack |
| Worker `stages:` / `workflows:` fields | Routing lives entirely in `orc.yaml` | Single source of truth; explicit errors when no worker assigned |
