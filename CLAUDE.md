# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**tmux-orc** is a multi-agent Claude orchestration system that uses tmux panes to run Claude Code agents in parallel. An orchestrator agent coordinates workers via an MCP server (message bus), with git worktrees providing repo isolation per worker.

This repo will be split into two separate deliverables:
- `tmux-claude-agents` — TPM-installable tmux plugin (bash)
- `claude-agents-mcp` — MCP server published to npm (TypeScript/Node)

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

### MCP Server (`claude-agents-mcp`)

A TypeScript/Node server that replaces file-based coordination. Exposes four tools to all agents:
- `get_task(worker_id)` — worker pulls its next assignment (pull-based/self-scheduling)
- `submit_result(worker_id, result)` — worker posts output when done
- `get_result(worker_id)` — orchestrator reads worker output
- `all_done()` — returns true when all workers have finished

### tmux Plugin (`tmux-claude-agents`)

```
tmux-claude-agents/
  tmux-claude-agents.tmux    # registers keybinds
  scripts/
    start_session.sh         # spin up orchestrator + worker panes
    start_mcp.sh             # start the MCP server
    dispatch.sh              # send-keys helper
    notify.sh                # macOS notifications
    cleanup.sh               # remove worktrees, kill MCP server
  templates/
    orchestrator.md          # default orchestrator prompt
    worker.md                # default worker bootstrap prompt
```

### Git Worktrees

Each worker creates and owns its own worktree lifecycle:
```bash
git worktree add .worktrees/worker2 -b agent/worker2
# ... do work ...
git worktree remove .worktrees/worker2
git branch -d agent/worker2
```

`.worktrees/` must be in `.gitignore`.

### Project Config (`.claude/agents.json`)

Custom config (not official Claude Code) the orchestrator reads to know worker roles and domain boundaries:
```json
{
  "workers": [
    { "id": 2, "role": "frontend", "domain": "src/frontend/", "stack": "React" },
    { "id": 3, "role": "backend", "domain": "src/backend/", "stack": "FastAPI" }
  ]
}
```

## Orchestrator Startup Sequence

1. Read `.claude/agents.json` for worker roles/domains
2. Start MCP server as background process
3. Spin up worker panes via `tmux send-keys` with role/domain in the initial prompt
4. Load tasks into MCP server
5. Workers self-bootstrap: create worktree → write CLAUDE.md → call `get_task()` → loop

## Worker Lifecycle

Workers are dispatched with a prompt like:
```
You are the frontend worker. Your domain is src/frontend/ (React).
First: create your worktree at .worktrees/worker2 on branch agent/worker2.
Then: write your CLAUDE.md into the worktree for reference.
Then: call get_task(worker_id=2) and begin.
```

Then the worker loops: `get_task() → do work → submit_result() → get_task() again`

## tmux Configuration

tmux.conf options for the plugin:
```tmux
set -g @plugin 'yourname/tmux-claude-agents'
set -g @claude-agents-workers 2
set -g @claude-agents-mcp-port 7777
set -g @claude-agents-notify true

# highlight panes on bell
set -g bell-action any
set -g visual-bell on
```

Keybind to launch a workspace (registered by plugin):
```tmux
bind M run-shell "~/.tmux/plugins/tmux-claude-agents/scripts/start_session.sh"
bind C-m run-shell "~/.tmux/plugins/tmux-claude-agents/scripts/cleanup.sh"
```

## macOS Notifications

Workers signal state via `osascript` with distinct sounds:
```bash
# done
osascript -e 'display notification "Worker 2 finished" with title "Claude Agent" sound name "Glass"'
# blocked
osascript -e 'display notification "Worker 2 is blocked" with title "Claude Agent" sound name "Basso"'
```

## Open Tasks

- [ ] Create repos and initial project structure
- [ ] Build the MCP server in TypeScript (`claude-agents-mcp`)
- [ ] Write the tmux plugin bash scripts
- [ ] Write the orchestrator and worker prompt templates
- [ ] Test with a small real task (e.g. auth feature with React + FastAPI)
