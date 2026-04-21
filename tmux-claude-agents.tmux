#!/usr/bin/env bash
PLUGIN_DIR="$(cd "$(dirname "$0")" && pwd)"

BUN=$(tmux show-option -gqv "@claude-agents-bun-path")
BUN="${BUN:-/opt/homebrew/bin/bun}"

MCP_PORT=$(tmux show-option -gqv "@claude-agents-mcp-port")
MCP_PORT="${MCP_PORT:-7777}"
export CLAUDE_AGENTS_MCP_PORT="$MCP_PORT"

WATCH_JOBS=$(tmux show-option -gqv "@claude-agents-watch-jobs")
WATCH_JOBS="${WATCH_JOBS:-false}"
export CLAUDE_AGENTS_WATCH_JOBS="$WATCH_JOBS"

# Warn once at plugin load if bun dependencies haven't been installed
if [ ! -d "$PLUGIN_DIR/mcp/node_modules" ]; then
  tmux display-message "tmux-claude-agents: run 'cd $PLUGIN_DIR/mcp && $BUN install' to finish setup"
fi

# Wrap a string in single quotes, escaping any embedded single quotes.
# This handles spaces and special characters in $BUN and $PLUGIN_DIR.
# All variables are expanded now (at plugin load time), not at key-press time.
sq() { printf "'%s'" "${1//\'/\'\\\'\'}"; }

CMD="$(sq "$BUN") run $(sq "$PLUGIN_DIR")/cli.ts"
PAUSE='; echo; read -r -s -n1'

tmux bind-key M   run-shell "$CMD launch"
tmux bind-key M-M run-shell "$CMD launch --here"
tmux bind-key S   run-shell "$CMD menu"
