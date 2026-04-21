# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**tmux-orc** is a multi-agent Claude orchestration system. An orchestrator agent coordinates workers via a local MCP server (message bus), with git worktrees providing per-job repo isolation. Workers run in tmux panes and are pull-based — they call `get_task` themselves when ready.

This repo will be split into two deliverables:
- `tmux-claude-agents` — TPM-installable tmux plugin (bash)
- `claude-agents-mcp` — MCP server published to npm (TypeScript/Bun)

## Architecture

```
┌──────────────────┬──────────────────┐
│  pane 1          │  pane 2          │
│  orchestrator    │  worker 1        │
│  (claude)        │  (claude)        │
│                  ├──────────────────┤
│                  │  pane 3          │
│                  │  worker 2        │
│                  │  (claude)        │
└──────────────────┴──────────────────┘
         ↕                  ↕
    MCP Server (local, port 7777 default)
```

### MCP Server (`mcp/`)

Bun/TypeScript HTTP/SSE server. Not stdio — runs persistently and serves multiple agents simultaneously. All state is in-memory (`state.ts`). Shared types live in `types.ts`.

Tools exposed to agents:

| Tool | Caller | Purpose |
|---|---|---|
| `register_worker(worker_id, pane_id)` | Worker | Register on startup |
| `get_task(worker_id, role)` | Worker | Pull next role-matched task |
| `submit_result(worker_id, result)` | Worker | Post output when done |
| `load_tasks(tasks[])` | Orchestrator | Seed the task queue |
| `get_result(worker_id)` | Orchestrator | Read a worker's result |
| `get_status()` | Orchestrator | Queue depth + worker states |
| `all_done()` | Orchestrator | True when queue empty and all workers are submitted or idle |
| `stage_done(job, stage)` | Orchestrator | True when all stage tasks submitted |
| `get_stage_results(job, stage)` | Orchestrator | All results from a stage |
| `get_jobs_status(job?)` | Orchestrator | Stage breakdown for one or all jobs |
| `reset_job(job)` | Orchestrator | Clear stage state to rerun a job |
| `get_hung_workers(threshold_ms)` | Orchestrator | Workers in working state with no activity for longer than threshold_ms |
| `report_blocked(worker_id, reason)` | Worker | Signal stuck state — fires Basso notification |
| `resolve_block(worker_id, resolution)` | Worker | Unblock self and append resolution to `## Lessons Learned` in `.claude/roles/<role>.md` |

HTTP inspection endpoints (curl-friendly): `/status`, `/queue`, `/results`, `/result/:worker`, `/jobs`, `/job/:name`, `/job/:name/:stage/results`

### Project Config

`.claude/agents.json` defines workers and pipeline definitions:
```json
{
  "workers": [
    { "id": "bob", "role": "frontend" },
    { "id": "rex", "role": "review" }
  ],
  "pipelines": [
    {
      "name": "frontend",
      "stages": [
        { "name": "build",  "role": "frontend" },
        { "name": "review", "role": "review"   }
      ]
    }
  ]
}
```

Jobs live as markdown files in `.claude/jobs/<name>.md` with YAML frontmatter (`pipeline:`, `domain:`) and a free-form spec body. Completed jobs are moved to `.claude/jobs/done/` with an `## Outcome` section appended.

### Git Worktrees

One shared worktree per job, created by the orchestrator:
```bash
git worktree add .worktrees/auth-login -b agent/auth-login
```
All workers in the same job share this worktree. After the final stage the orchestrator removes the worktree; the branch stays for the open PR.

`.worktrees/` must be in `.gitignore`.

## CLI

`cli.ts` at the project root is the primary entry point (called by all tmux keybinds):

```
bun run cli.ts init       # scaffold .claude/ in a new project (agents.json, sample job, dirs)
bun run cli.ts validate   # pre-flight checks: roles, skills, plugins, job frontmatter
bun run cli.ts start      # starts MCP server + orchestrator; --job=<name> spawns workers too
bun run cli.ts start-mcp  # launches bun server, guards double-start via PID file
bun run cli.ts watch      # watches .claude/jobs/ and auto-starts new jobs
bun run cli.ts menu       # tmux display-menu for status inspection
bun run cli.ts cleanup    # kills MCP server, removes worktrees + branches
bun run cli.ts notify     # macOS notifications (Glass = done, Basso = blocked)
```

`start --job=<name>` creates the worktree, installs skills, spawns one pane per worker (with role content injected into the bootstrap prompt), then starts the orchestrator. Without `--job=`, only the orchestrator pane is created and it handles worker setup manually.

The bash scripts in `scripts/` are kept as backup and are not invoked directly.

## Worker Lifecycle

```
register_worker() → get_task() → do work → submit_result() → get_task() → ...
```

When `get_task` returns `NO_TASKS`, workers wait 30 seconds and retry. They stay alive for the full session and pick up new jobs as they are loaded.

## Running Tests

```bash
bun test
```

## tmux Configuration

```tmux
set -g @plugin 'yourname/tmux-claude-agents'
set -g @claude-agents-mcp-port   7777   # default
set -g @claude-agents-notify     true   # macOS notifications (set false to disable)
set -g @claude-agents-watch-jobs true   # auto-start jobs dropped into .claude/jobs/

set -g bell-action any
set -g visual-bell on
```
