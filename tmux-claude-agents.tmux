#!/usr/bin/env bash
PLUGIN_DIR="$(cd "$(dirname "$0")" && pwd)"

MCP_PORT=$(tmux show-option -gqv "@claude-agents-mcp-port")
MCP_PORT="${MCP_PORT:-7777}"
export CLAUDE_AGENTS_MCP_PORT="$MCP_PORT"

WATCH_JOBS=$(tmux show-option -gqv "@claude-agents-watch-jobs")
WATCH_JOBS="${WATCH_JOBS:-false}"
export CLAUDE_AGENTS_WATCH_JOBS="$WATCH_JOBS"

tmux bind-key M run-shell "bun run \"$PLUGIN_DIR/cli.ts\" start"
tmux bind-key M-m run-shell "bun run \"$PLUGIN_DIR/cli.ts\" start --here"
tmux bind-key S run-shell "bun run \"$PLUGIN_DIR/cli.ts\" menu"
tmux bind-key C-m run-shell "bun run \"$PLUGIN_DIR/cli.ts\" cleanup"
