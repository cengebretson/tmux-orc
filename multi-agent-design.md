# Multi-Agent Claude + tmux Design

## Concept
Use tmux panes to run multiple Claude Code agents in parallel, with full visibility
and the ability to intervene. An orchestrator agent coordinates workers via an MCP
server acting as a message bus, with git worktrees providing isolation.

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
```

## Key Components

### MCP Server (message bus)
A small local Bun/TypeScript server replacing file-based coordination.
Exposes tools to all agents:
- `get_task(worker_id)` — worker pulls its assignment
- `submit_result(worker_id, result)` — worker posts output when done
- `get_result(worker_id)` — orchestrator reads worker output
- `all_done()` — returns true when all workers finished

Workers are pull-based — they call `get_task()` themselves when ready,
making the system self-scheduling.

### Git Worktrees

Worktree ownership depends on the task mode:

**Pipeline** — one shared worktree per pipeline, created by the orchestrator before dispatching workers:
```bash
git worktree add .worktrees/auth -b agent/auth
```
All workers in the pipeline share this worktree and branch. The review worker sees the build worker's commits immediately. The git worker opens a single PR from `agent/auth` → `main` with everything on one branch — no cross-branch merging.

**Standalone** — one worktree per worker, also created by the orchestrator:
```bash
git worktree add .worktrees/bob   -b agent/bob
git worktree add .worktrees/alice -b agent/alice
```

Cleanup after merge:
```bash
# pipeline
git worktree remove .worktrees/auth
git branch -d agent/auth

# standalone
git worktree remove .worktrees/bob
git branch -d agent/bob
```

### Project Config
`.claude/agents.json` (custom, not official Claude Code):
```json
{
  "workers": [
    { "id": "bob-the-webdev", "role": "frontend", "domain": ["src/frontend/", "src/shared/"] },
    { "id": "alice-the-api",  "role": "backend",  "domain": "src/backend/" }
  ]
}
```
Orchestrator reads this to know domain boundaries when writing worker CLAUDE.md files.

### CLAUDE.md Files
- Root `CLAUDE.md` — shared conventions, project structure, coordination protocol
- Per-worktree `CLAUDE.md` — written by the worker itself during bootstrap

Add to `.gitignore` to keep agent-generated files out of commits:
```
.worktrees/
```

## Orchestrator Startup Sequence
1. Read `.claude/agents.json` for worker roles/domains
2. Start MCP server as background process
3. Spin up worker panes via `tmux send-keys` with role/domain in the initial prompt
4. Load tasks into MCP server
5. Workers self-bootstrap and call `get_task()` when ready

## Worker Bootstrap (first actions on startup)

The orchestrator creates all worktrees before dispatching workers, then pastes each worker's bootstrap prompt via tmux. The prompt tells the worker whether a worktree is already set up (pipeline) or needs creating (standalone).

**Pipeline worker prompt:**
```
You are the frontend worker. Your pipeline worktree is already set up at
.worktrees/auth on branch agent/auth. Do not create a new worktree.
Call register_worker, then get_task(worker_id="bob", role="frontend") and begin.
```

**Standalone worker prompt:**
```
You are the frontend worker. Your domain is src/frontend/ (React).
Create your worktree: git worktree add .worktrees/bob -b agent/bob
Then call register_worker, then get_task(worker_id="bob", role="frontend") and begin.
```

## Worker Lifecycle
```
receive initial prompt → [create worktree if standalone] → register_worker() → call get_task() → do work → call submit_result() → call get_task() again
```

## macOS Notifications
Workers alert you when done or blocked:
```bash
# done
osascript -e 'display notification "Worker 2 finished" with title "Claude Agent" sound name "Glass"'

# blocked / needs input
osascript -e 'display notification "Worker 2 is blocked" with title "Claude Agent" sound name "Basso"'
```
Different sounds = know what needs attention without looking.

Also configure tmux to highlight panes on bell:
```tmux
set -g bell-action any
set -g visual-bell on
```

## tmux Keybind (not yet added to config)
Spin up a fresh multi-agent workspace:
```tmux
bind M new-window -n "multi-agent" \; \
  split-window -h -c "#{pane_current_path}" \; \
  split-window -v -t right -c "#{pane_current_path}" \; \
  send-keys -t 1 "claude" Enter \; \
  send-keys -t 2 "claude" Enter \; \
  send-keys -t 3 "claude" Enter \; \
  select-pane -t 1
```

## Packaging: tmux Plugin

The whole system is a single TPM-installable tmux plugin. The MCP server ships as
TypeScript source inside the plugin and is run directly by Bun — no separate npm
package or publish step. Bun is the only external runtime dependency (`brew install bun`).

```
tmux-claude-agents/
  tmux-claude-agents.tmux    # main plugin file, registers keybinds
  mcp/
    server.ts                # MCP server (Bun/TypeScript, runs in place)
    package.json             # MCP SDK dependency only
  scripts/
    start_session.sh         # spin up orchestrator + worker panes
    start_mcp.sh             # bun run mcp/server.ts --port $MCP_PORT
    dispatch.sh              # send-keys helper
    notify.sh                # macOS notifications
    cleanup.sh               # remove worktrees, kill MCP server
  templates/
    orchestrator.md          # default orchestrator prompt
    worker.md                # default worker bootstrap prompt
```

Users install via TPM:
```tmux
set -g @plugin 'yourname/tmux-claude-agents'
```

Configure in `tmux.conf`:
```tmux
set -g @claude-agents-workers 2
set -g @claude-agents-mcp-port 7777
set -g @claude-agents-notify true
```

Keybinds registered by the plugin:
```bash
# tmux-claude-agents.tmux
tmux bind-key M run-shell "~/.tmux/plugins/tmux-claude-agents/scripts/start_session.sh"
tmux bind-key C-m run-shell "~/.tmux/plugins/tmux-claude-agents/scripts/cleanup.sh"
```

## MCP Server

The plugin's `start_mcp.sh` runs the server directly:
```bash
bun run ~/.tmux/plugins/tmux-claude-agents/mcp/server.ts --port ${MCP_PORT:-7777}
```

No global install or publish step needed. Updating the plugin updates the server.

### Transport: HTTP/SSE

The MCP server uses HTTP/SSE transport (not stdio) so it can run persistently in the
background and serve multiple agents simultaneously.

### MCP URL Injection via tmux Environment

The plugin owns the wiring: since `start_session.sh` knows the port (from
`@claude-agents-mcp-port`), it injects the URL as an environment variable into each
pane at creation time:

```bash
# start_session.sh
MCP_URL="http://localhost:${MCP_PORT:-7777}"
tmux new-window -n "multi-agent" -e "MCP_URL=${MCP_URL}"
```

Each agent's bootstrap prompt tells it to register the MCP server on startup:
```
Your MCP server is available at $MCP_URL — run `claude mcp add` to connect it.
```

This keeps prompt templates generic (no hardcoded URLs) and means agents don't need
any prior MCP configuration — the plugin handles all the wiring.

## Advantages Over Native Subagents
- Full visibility — watch each agent work in real time
- Can intervene, redirect, or correct mid-task
- Agents persist across long-running interactive sessions
- MCP server gives structured coordination vs file polling

## Skills and Plugins

During worker bootstrap the orchestrator copies skill files into each worker's worktree
at `.worktrees/<id>/.claude/commands/`, making them available as slash commands.

Lookup order (same pattern as roles):
1. `.claude/skills/<skill>.md` — project-level, takes precedence
2. `~/.tmux/plugins/tmux-claude-agents/skills/<skill>.md` — plugin built-ins, fallback

Workers have access to all installed skills — there is no per-worker filtering. Each
role file documents which skills and plugins that role should use via `## Skills` and
`## Plugins` sections. Workers read this from their CLAUDE.md and know what tools are
relevant without any enforcement machinery.

This is intentionally simple. Per-worker skill/plugin scoping can be added later if
needed — the role file approach gives clear guidance with zero infrastructure overhead.

## Task Modes

There are two ways to run tasks, chosen per session:

### Standalone (parallel, independent)

Tasks have no `pipeline` or `stage` fields. Workers pull tasks by role and run in
parallel with no ordering dependency. The orchestrator monitors via:
- `all_done(workerCount)` — returns true when queue is empty and all workers submitted
- `get_result(workerId)` — reads a single worker's output

Use this when tasks are independent: e.g. multiple isolated components being built
simultaneously.

### Pipeline (sequential stages, results feed forward)

Tasks carry three extra fields:
- `pipeline` — the reusable definition name (e.g. `"frontend"`) — label only
- `job` — the specific feature run (e.g. `"auth-login"`) — coordination key
- `stage` — the stage within the job (e.g. `"build"`)

Results are automatically attributed to the correct job/stage when a worker calls
`submit_result`. The orchestrator sequences stages via:
- `stage_done(job, stage)` — poll until all tasks in a stage are submitted
- `get_stage_results(job, stage)` — read all results from a completed stage, then
  build and load the next stage's tasks

Use this when work must happen in order: e.g. build → review → security → ship.

Multiple jobs can run the same pipeline simultaneously — each job gets its own
worktree, branch, and independent stage state on the server.

All tasks can be loaded up front with `load_tasks` — workers self-schedule by role,
pulling only tasks that match their role from the queue.

---

## Pipelines and Jobs

A **pipeline** is a reusable definition of stages and roles. A **job** is a specific
execution of a pipeline for a particular feature. The separation means the same
pipeline can run twice in parallel for two different features simultaneously.

`agents.json` defines the workforce and available pipeline definitions:

```json
{
  "workers": [
    { "id": "bob", "role": "frontend" },
    { "id": "rex", "role": "review"   },
    { "id": "sam", "role": "security" },
    { "id": "git", "role": "git"      }
  ],
  "pipelines": [
    {
      "name": "frontend",
      "stages": ["build", "review", "security", "ship"]
    }
  ]
}
```

The orchestrator creates jobs at runtime by loading tasks tagged with `pipeline`,
`job`, and `stage`. Domain belongs to the job, not the pipeline definition:

```json
[
  { "id": "1", "role": "frontend", "pipeline": "frontend", "job": "auth-login", "stage": "build", "domain": "src/frontend/auth/" },
  { "id": "2", "role": "frontend", "pipeline": "frontend", "job": "dashboard",  "stage": "build", "domain": "src/frontend/dashboard/" }
]
```

The orchestrator manages sequencing using two MCP tools:
- `stage_done(job, stage)` — poll until a stage is complete
- `get_stage_results(job, stage)` — read results to build the next stage's tasks

Stages with multiple dependencies (e.g. `ship` depends on both `review` and
`security`) run their dependencies in parallel and wait for both before proceeding.

Stage status is tracked automatically — when a worker submits a result the server
attributes it to the correct job/stage via the worker's current task.

## Communication Rules (Hub-and-Spoke)

All inter-agent communication routes through the orchestrator — workers never talk directly
to each other. This prevents broadcast storms and keeps coordination predictable.

- Workers report results only via `submit_result` → orchestrator reads via `get_result`
- If worker B needs worker A's output, the orchestrator reads A's result and passes the
  relevant parts as a new task to B
- n workers means n communication channels (to orchestrator), not n² between workers

## Shared Knowledge: CLAUDE.md as Shared Brain
All agents (native subagents or tmux panes) automatically load CLAUDE.md.
Define coordination protocol once there — every agent knows the rules.

## Project Location
One repo: `tmux-claude-agents` — the tmux plugin (bash + Bun/TypeScript MCP server).
To be built in a separate repo, not in the tmux config folder.

## Open Questions / Next Steps
- [ ] Create repo and initial project structure
- [ ] Build the MCP server in TypeScript (mcp/server.ts, run via Bun)
- [ ] Write the tmux plugin bash scripts
- [ ] Write the orchestrator prompt template
- [ ] Write the worker bootstrap prompt template
- [ ] Add tmux-logging plugin for full output capture
- [ ] Test with a small real task (e.g. auth feature with React + FastAPI)
