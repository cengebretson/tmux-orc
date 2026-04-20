#!/usr/bin/env bash
PLUGIN_DIR="$(cd "$(dirname "$0")" && pwd)"

MCP_PORT=$(tmux show-option -gqv "@claude-agents-mcp-port")
MCP_PORT="${MCP_PORT:-7777}"

export CLAUDE_AGENTS_MCP_PORT="$MCP_PORT"

tmux bind-key M run-shell "\"$PLUGIN_DIR/scripts/start_session.sh\""
tmux bind-key M-m run-shell "\"$PLUGIN_DIR/scripts/start_session.sh\" --here"
tmux bind-key S run-shell "\"$PLUGIN_DIR/scripts/menu.sh\""
tmux bind-key C-m run-shell "\"$PLUGIN_DIR/scripts/cleanup.sh\""
