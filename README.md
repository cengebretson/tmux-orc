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
    { "id": "bob-the-webdev", "role": "frontend", "domain": ["src/frontend/", "src/shared/"] },
    { "id": "alice-the-api",  "role": "backend",  "domain": "src/backend/" }
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

Built-in roles: `backend`, `frontend`, `code-review`, `documentation-writer`

Workers only receive tasks matching their role.

### Adding a custom role

Create a markdown file in the plugin's `roles/` directory named after the role:

```bash
# ~/.tmux/plugins/tmux-claude-agents/roles/data-engineer.md
```

The file defines the worker's persona, expertise, and standards. Then use the role name in `agents.json` and tasks:

```json
{ "id": "dave-the-data", "role": "data-engineer", "domain": "pipelines/" }
```

If a role is used in `agents.json` but has no matching file in `roles/`, the session will fail to start with a clear error.

### Inspection Endpoints

The MCP server also exposes plain HTTP endpoints for quick inspection:

```bash
curl http://localhost:7777/status          # queue depth + worker states
curl http://localhost:7777/queue           # pending tasks
curl http://localhost:7777/results         # all submitted results
curl http://localhost:7777/result/2        # result for worker 2
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

Each worker operates on its own git worktree and branch:

```bash
git worktree add .worktrees/worker2 -b agent/worker2
```

Workers create and clean up their own worktrees — the orchestrator doesn't manage them.

### Communication rules

- Workers report only to the orchestrator via `submit_result`, never to each other
- If worker B needs worker A's output, the orchestrator reads A's result and passes the relevant parts as a new task to B

## Running tests

```bash
cd mcp && bun test
```
