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
Each worker gets an isolated copy of the repo on its own branch:
```bash
git worktree add .worktrees/worker2 -b agent/worker2
git worktree add .worktrees/worker3 -b agent/worker3
```
Cleanup after merge:
```bash
git worktree remove .worktrees/worker2
git worktree remove .worktrees/worker3
git branch -d agent/worker2 agent/worker3
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
The orchestrator dispatches each worker with its role and domain in the initial
`send-keys` message. The worker then self-bootstraps:

```
You are the frontend worker. Your domain is src/frontend/ (React).
First: create your worktree at .worktrees/worker2 on branch agent/worker2.
Then: write your CLAUDE.md into the worktree for reference.
Then: call get_task(worker_id=2) and begin.
```

Worker's first steps:
```bash
git worktree add .worktrees/worker2 -b agent/worker2
# write own CLAUDE.md into worktree for persistent reference
# call get_task(worker_id=2)
```

This avoids the orchestrator needing to manage worktree creation — each worker
owns its full lifecycle from creation to cleanup.

## Worker Lifecycle
```
receive initial prompt → create worktree → write CLAUDE.md → call get_task() → do work → call submit_result() → notify orchestrator → call get_task() again
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

Tasks carry `pipeline` and `stage` fields. Results are automatically attributed to the
correct stage when a worker calls `submit_result`. The orchestrator sequences stages via:
- `stage_done(pipeline, stage)` — poll until all tasks in a stage are submitted
- `get_stage_results(pipeline, stage)` — read all results from a completed stage, then
  build and load the next stage's tasks

Use this when work must happen in order: e.g. build → review → security → ship.

All tasks can be loaded up front with `load_tasks` — workers self-schedule by role,
pulling only tasks that match their role from the queue.

---

## Pipelines

Pipelines define sequential stage-based workflows where each stage's results feed the
next. Domain belongs to the pipeline, not the worker — the same role can participate
in multiple pipelines across different domains.

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

The orchestrator manages sequencing using two MCP tools:
- `stage_done(pipeline, stage)` — poll until a stage is complete
- `get_stage_results(pipeline, stage)` — read results to build the next stage's tasks

Stages with multiple `input` entries (e.g. `ship` depends on both `review` and
`security`) run their dependencies in parallel and wait for both before proceeding.

Stage status is tracked automatically — when a worker submits a result the server
attributes it to the correct pipeline/stage via the worker's current task.

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
