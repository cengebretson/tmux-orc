# tmux-claude-agents

```
⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢀⡀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢠⣿⣿⡄⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠀⠀⠀⣀⣤⣶⣧⣄⣉⣉⣠⣼⣶⣤⣀⠀⠀⠀⠀⠀⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠀⢰⣿⣿⣿⣿⡿⣿⣿⣿⣿⢿⣿⣿⣿⣿⡆⠀⠀⠀⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠀⣼⣤⣤⣈⠙⠳⢄⣉⣋⡡⠞⠋⣁⣤⣤⣧⠀⠀⠀⠀⠀⠀⠀
⠀⢲⣶⣤⣄⡀⢀⣿⣄⠙⠿⣿⣦⣤⡿⢿⣤⣴⣿⠿⠋⣠⣿⠀⢀⣠⣤⣶⡖⠀
⠀⠀⠙⣿⠛⠇⢸⣿⣿⡟⠀⡄⢉⠉⢀⡀⠉⡉⢠⠀⢻⣿⣿⡇⠸⠛⣿⠋⠀⠀
⠀⠀⠀⠘⣷⠀⢸⡏⠻⣿⣤⣤⠂⣠⣿⣿⣄⠑⣤⣤⣿⠟⢹⡇⠀⣾⠃⠀⠀⠀
⠀⠀⠀⠀⠘⠀⢸⣿⡀⢀⠙⠻⢦⣌⣉⣉⣡⡴⠟⠋⡀⢀⣿⡇⠀⠃⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⢸⣿⣧⠈⠛⠂⠀⠉⠛⠛⠉⠀⠐⠛⠁⣼⣿⡇⠀⠀⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠸⣏⠀⣤⡶⠖⠛⠋⠉⠉⠙⠛⠲⢶⣤⠀⣹⠇⠀⠀⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠀⠀⠀⢹⣿⣶⣿⣿⣿⣿⣿⣿⣶⣿⡏⠀⠀⠀⠀⠀⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠀⠀⠀⠈⠉⠉⠉⠛⠛⠛⠛⠉⠉⠉⠁⠀⠀⠀⠀⠀⠀⠀⠀⠀
```

A tmux plugin for running multiple Claude Code agents in parallel. An orchestrator agent coordinates workers via a local MCP server acting as a message bus, with git worktrees providing per-worker repo isolation.

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
    MCP Server (localhost:7777)
```

Workers are **pull-based** — they call `get_task` themselves when ready, making the system self-scheduling. All inter-agent communication routes through the orchestrator (hub-and-spoke), never directly between workers.

## Requirements

- [tmux](https://github.com/tmux/tmux) 3.2+
- [Claude Code](https://claude.ai/code) CLI (`claude`)
- [Bun](https://bun.sh) (`brew install bun`)
- [jq](https://jqlang.github.io/jq/) (`brew install jq`)

## Installation

### Via TPM

Add to your `tmux.conf`:

```tmux
set -g @plugin 'yourname/tmux-claude-agents'
```

Then press `prefix+I` to install.

### Manual

```bash
git clone https://github.com/yourname/tmux-claude-agents ~/.tmux/plugins/tmux-claude-agents
~/.tmux/plugins/tmux-claude-agents/tmux-claude-agents.tmux
```

Install MCP server dependencies:

```bash
cd ~/.tmux/plugins/tmux-claude-agents/mcp && bun install
```

### Configuration

Add to `tmux.conf`:

```tmux
set -g @claude-agents-mcp-port 7777   # default
set -g @claude-agents-notify  true    # macOS notifications
```

## Project Setup

Create `.claude/agents.json` in your project repo:

```json
{
  "workers": [
    { "id": "bob",  "role": "frontend" },
    { "id": "rex",  "role": "review"   },
    { "id": "sam",  "role": "security" },
    { "id": "git",  "role": "git"      }
  ],
  "pipelines": [
    {
      "name": "auth-feature",
      "domain": ["src/frontend/auth/", "src/backend/auth/"],
      "stages": [
        { "stage": "build",    "role": "frontend" },
        { "stage": "review",   "role": "review",   "input": "build" },
        { "stage": "security", "role": "security", "input": "build" },
        { "stage": "ship",     "role": "git",      "input": ["review", "security"] }
      ]
    }
  ]
}
```

Add to `.gitignore`:

```
.worktrees/
```

## Usage

| Keybind | Action |
|---|---|
| `prefix+M` | Start a multi-agent session (new window) |
| `prefix+Alt+M` | Start in the current pane |
| `prefix+S` | Open status menu |
| `prefix+Ctrl+M` | Clean up — kill MCP server, remove worktrees |

### Starting a session

Press `prefix+M` from inside your project directory. The plugin will:

1. Start the MCP server in the background
2. Open a new window and launch the orchestrator Claude agent
3. The orchestrator reads `agents.json`, creates worker panes, and sends each worker its bootstrap prompt

### What the orchestrator does

1. Registers the MCP server and loads tasks via `load_tasks`
2. Workers spin up, create their own git worktrees, and start pulling tasks
3. Monitor progress with `get_status` or `prefix+S`
4. When `all_done` returns true, aggregate results with `get_result`

## Example: Auth feature pipeline

This walkthrough shows a full pipeline session: four workers, four stages, results feeding forward.

### Project config

`.claude/agents.json`:

```json
{
  "workers": [
    { "id": "bob",  "role": "frontend" },
    { "id": "rex",  "role": "review"   },
    { "id": "sam",  "role": "security" },
    { "id": "git",  "role": "git"      }
  ]
}
```

### Step 1 — Start the session

Press `prefix+M` from your project directory. The plugin starts the MCP server and opens a new `agents` window with the orchestrator:

```
┌──────────────────────────────────────────┐
│  agents window                           │
│                                          │
│  > claude                                │
│  [orchestrator reading prompt...]        │
│                                          │
└──────────────────────────────────────────┘
```

### Step 2 — Orchestrator creates the pipeline worktree

Before spinning up any workers, the orchestrator creates a single shared worktree for the pipeline:

```bash
git worktree add .worktrees/auth -b agent/auth
```

All pipeline workers (`bob`, `rex`, `sam`, `git`) will work inside `.worktrees/auth` on branch `agent/auth`. This means the review worker sees bob's commits immediately, and the git worker opens one PR from `agent/auth` → `main`.

### Step 3 — Orchestrator spins up workers

The orchestrator creates a pane per worker, writes the role file as `CLAUDE.md` into `.worktrees/auth`, installs skills there, and pastes each worker's bootstrap prompt:

```
┌─────────────────────┬────────────────────┐
│  orchestrator       │  bob (frontend)    │
│                     │  > claude          │
│  "Spinning up       ├────────────────────┤
│   workers..."       │  rex (review)      │
│                     │  > claude          │
│                     ├────────────────────┤
│                     │  sam (security)    │
│                     │  > claude          │
└─────────────────────┴────────────────────┘
```

### Step 4 — Workers self-bootstrap

Each worker independently runs its bootstrap loop. Because it's a pipeline session, the worktree already exists — workers just register and start pulling tasks:

```
register_worker(worker_id="bob", pane_id="%23")
# worktree .worktrees/auth already exists — no git worktree add needed
get_task(worker_id="bob", role="frontend")
→ task p1: "Build auth login form"
```

### Step 5 — Orchestrator loads the pipeline

```json
load_tasks([
  { "id": "p1", "role": "frontend", "description": "Build login + signup forms with JWT token handling", "pipeline": "auth", "stage": "build",    "domain": "src/frontend/auth/" },
  { "id": "p2", "role": "review",   "description": "Review the auth frontend changes",                   "pipeline": "auth", "stage": "review"   },
  { "id": "p3", "role": "security", "description": "Audit the auth flow for vulnerabilities",             "pipeline": "auth", "stage": "security" },
  { "id": "p4", "role": "git",      "description": "Open a PR merging agent/bob into main",               "pipeline": "auth", "stage": "ship"     }
])
```

All tasks are loaded at once. Workers self-schedule by role — bob picks up `p1` immediately since he already called `get_task`. Rex and Sam are waiting; their tasks are in the queue and will be claimed when they call `get_task`.

### Step 6 — Build stage

Bob works in `.worktrees/bob`. The orchestrator polls:

```
stage_done(pipeline="auth", stage="build") → false ... false ... true ✓
```

Bob submits his result:

```
submit_result(worker_id="bob", result="Login + signup forms complete. JWT stored in httpOnly cookie. Files: src/frontend/auth/Login.tsx, Signup.tsx, useAuth.ts")
```

### Step 7 — Review + security run in parallel

Both `review` and `security` depend only on `build`, so they run simultaneously. Rex and Sam both call `get_task` and pick up their tasks. The orchestrator polls both:

```
stage_done("auth", "review")   → false ... true ✓
stage_done("auth", "security") → false ... false ... true ✓
```

```
┌─────────────────────┬────────────────────┐
│  orchestrator       │  bob (frontend)    │
│                     │  ✓ submitted       │
│  "build done,       ├────────────────────┤
│   review+security   │  rex (review)      │
│   running..."       │  [reading diff...] │
│                     ├────────────────────┤
│                     │  sam (security)    │
│                     │  [auditing auth..] │
└─────────────────────┴────────────────────┘
```

### Step 8 — Ship stage

Both stages done. Orchestrator reads their results and loads the final task with the findings baked into the description:

```
get_stage_results("auth", "review")   → { "rex": "LGTM, 2 minor comments..." }
get_stage_results("auth", "security") → { "sam": "No critical issues. CSRF token missing on signup form." }
```

```json
load_tasks([{
  "id": "p4",
  "role": "git",
  "description": "Open PR: agent/bob → main. Review notes: LGTM, 2 minor comments. Security: add CSRF token to signup form before merging.",
  "pipeline": "auth",
  "stage": "ship"
}])
```

The `git` worker picks it up, applies the CSRF fix, runs `/pr-description`, and opens a PR from `agent/auth` → `main`. All the pipeline's work is already on one branch — no merging needed.

### Step 9 — Done

```
stage_done("auth", "ship") → true
```

macOS notification fires: **"Worker git finished"** (Glass sound).

The orchestrator removes the pipeline worktree — the git worker has already committed everything to the branch:

```bash
git worktree remove .worktrees/auth
# branch agent/auth stays alive for the open PR
# delete it manually after the PR is merged: git branch -d agent/auth
```

Press `prefix+Ctrl+M` to kill the MCP server. If any worktrees are still around (e.g. session was aborted), cleanup will force-remove them and warn about any open branches.

### Inspect anytime

While the session runs, press `prefix+S` for the status menu or query the API directly:

```bash
curl localhost:7777/status                        # all worker states
curl localhost:7777/pipeline/auth                 # stage breakdown
curl localhost:7777/pipeline/auth/build/results   # bob's build output
```

## Status Menu

Press `prefix+S` to open the status menu. Worker entries are populated dynamically from the current session:

```
 Claude Agents 
  Status  s    ← queue depth + all worker states
  Queue   q    ← pending tasks
  Results r    ← all submitted results
  ──────────
  Worker 2     ← result for worker 2
  Worker 3     ← result for worker 3
```

Each option opens a tmux popup with formatted JSON output.

## MCP Server

The MCP server runs as a local HTTP/SSE server (Bun/TypeScript) bundled inside the plugin — no separate install needed.

### MCP Tools (used by agents)

| Tool | Called by | Description |
|---|---|---|
| `register_worker(worker_id, pane_id)` | Worker | Registers pane ID for health checking |
| `load_tasks(tasks[])` | Orchestrator | Seeds the task queue |
| `get_task(worker_id, role)` | Worker | Pulls next role-matched task |
| `submit_result(worker_id, result)` | Worker | Posts completed output |
| `get_result(worker_id)` | Orchestrator | Reads a worker's result |
| `get_status()` | Orchestrator | Queue depth + all worker states |
| `all_done(worker_count)` | Orchestrator | True when all workers have submitted |
| `stage_done(pipeline, stage)` | Orchestrator | True when all tasks in a stage are submitted |
| `get_stage_results(pipeline, stage)` | Orchestrator | All results from a completed stage |

### Tasks

Tasks are structured objects:

```json
{
  "id": "auth-1",
  "role": "backend",
  "description": "Implement JWT login endpoint",
  "domain": "src/backend/"
}
```

Built-in roles: `backend`, `frontend`, `review`, `document`, `security`, `git`

Workers only receive tasks matching their role.

#### Task modes

There are two ways to use tasks, chosen per session:

**Standalone** — tasks are independent, can run in any order. No `pipeline` or `stage` fields. Orchestrator monitors with `all_done(workerCount)` and reads results via `get_result(workerId)`.

```json
[
  { "id": "1", "role": "frontend", "description": "Build login form" },
  { "id": "2", "role": "backend",  "description": "Build login API"  }
]
```

**Pipeline** — tasks belong to named stages that run in sequence. Results from one stage feed the next. Tasks carry `pipeline` and `stage` fields; result attribution is automatic.

```json
[
  { "id": "p1", "role": "frontend", "description": "Build login form", "pipeline": "auth", "stage": "build"    },
  { "id": "p2", "role": "review",   "description": "Review auth PR",   "pipeline": "auth", "stage": "review"   },
  { "id": "p3", "role": "security", "description": "Security audit",   "pipeline": "auth", "stage": "security" },
  { "id": "p4", "role": "git",      "description": "Open PR",          "pipeline": "auth", "stage": "ship"     }
]
```

Orchestrator polls `stage_done(pipeline, stage)`, then reads `get_stage_results(pipeline, stage)` to build the next stage's tasks. Stages with multiple inputs (e.g. `ship` after both `review` and `security`) run their inputs in parallel and wait for both.

### Skills and plugins

Workers have access to all skills (`.claude/commands/`) and MCP plugins configured for the project — there is no per-worker filtering. Each role file documents which skills and plugins that role should use via `## Skills` and `## Plugins` sections. Workers read this from their CLAUDE.md and know what tools are relevant to them.

During bootstrap the orchestrator copies skill files into each worker's worktree at `.worktrees/<id>/.claude/commands/`, making them available as slash commands. The lookup order mirrors roles:

1. `.claude/skills/<skill>.md` — project-level, takes precedence
2. `~/.tmux/plugins/tmux-claude-agents/skills/<skill>.md` — plugin built-ins, fallback

Built-in skills (shipped in `skills/`):

| Skill | Description |
|---|---|
| `/component-review` | Self-review a React component before submitting |
| `/accessibility-check` | Check UI for keyboard, screen reader, and contrast issues |
| `/api-review` | Review an API endpoint for validation, auth, and error handling |
| `/test-coverage` | Assess test coverage by reading source and test files |
| `/security-review` | Security-focused pass covering injection, auth, secrets, and config |
| `/doc-review` | Review documentation for accuracy, clarity, and completeness |
| `/dependency-audit` | Audit dependencies for known vulnerabilities and abandoned packages |
| `/pr-description` | Generate a structured PR description from the current branch diff |

### Adding a custom role

Role files are looked up in this order:

1. `.claude/roles/<role>.md` — project-level, takes precedence
2. `~/.tmux/plugins/tmux-claude-agents/roles/<role>.md` — plugin built-ins, fallback

To add a project-specific role, create it alongside your `agents.json`:

```
your-project/
  .claude/
    agents.json
    roles/
      data-engineer.md   ← project-specific role
```

The file defines the worker's persona, expertise, and standards. Then reference it in `agents.json`:

```json
{ "id": "dave-the-data", "role": "data-engineer", "domain": "pipelines/" }
```

If a role is used in `agents.json` but has no matching file in either location, the session will fail to start with a clear error.

### Inspection Endpoints

The MCP server also exposes plain HTTP endpoints for quick inspection:

```bash
curl http://localhost:7777/status                          # queue depth + worker states
curl http://localhost:7777/queue                           # pending tasks
curl http://localhost:7777/results                         # all submitted results
curl http://localhost:7777/result/bob                      # result for a specific worker
curl http://localhost:7777/pipelines                       # all pipeline statuses
curl http://localhost:7777/pipeline/auth-feature           # stage breakdown for one pipeline
curl http://localhost:7777/pipeline/auth-feature/build/results  # results from a stage
```

## Architecture

```
tmux-claude-agents/
  tmux-claude-agents.tmux    # plugin entry point, registers keybinds
  mcp/
    server.ts                # HTTP entry point, routes MCP + inspection traffic
    mcp.ts                   # MCP server instance + tool registrations
    routes.ts                # inspection GET handlers
    state.ts                 # task queue, worker state, results (in-memory)
    state.test.ts            # unit tests (bun test)
    package.json
  scripts/
    start_session.sh         # starts MCP server, creates orchestrator pane
    start_mcp.sh             # launches bun server, guards double-start via PID
    menu.sh                  # tmux display-menu for status inspection
    cleanup.sh               # kills MCP server, removes worktrees + branches
    notify.sh                # macOS notifications (Glass = done, Basso = blocked)
  templates/
    orchestrator.md          # bootstrap prompt for the orchestrator agent
    worker.md                # bootstrap prompt for each worker agent
```

### Worker isolation

Worktree ownership depends on the task mode:

**Pipeline** — all workers in a pipeline share one worktree and branch, created by the orchestrator before dispatching:

```bash
git worktree add .worktrees/auth -b agent/auth
```

Sharing one branch means the review worker immediately sees the build worker's commits, and the git worker opens a single PR (`agent/auth` → `main`) with all the work on one branch.

**Standalone** — each worker gets its own worktree and branch, also created by the orchestrator:

```bash
git worktree add .worktrees/bob -b agent/bob
```

### Communication rules

- Workers report only to the orchestrator via `submit_result`, never to each other
- If worker B needs worker A's output, the orchestrator reads A's result and passes the relevant parts as a new task to B

## Running tests

```bash
cd mcp && bun test
```
