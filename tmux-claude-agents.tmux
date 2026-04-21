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

NOTIFY=$(tmux show-option -gqv "@claude-agents-notify")
NOTIFY="${NOTIFY:-true}"
export CLAUDE_AGENTS_NOTIFY="$NOTIFY"

LAYOUT=$(tmux show-option -gqv "@claude-agents-layout")
LAYOUT="${LAYOUT:-windows}"
export CLAUDE_AGENTS_LAYOUT="$LAYOUT"

# Auto-install dependencies on first use (bun install is fast with cache)
if [ ! -d "$PLUGIN_DIR/node_modules" ] || [ ! -d "$PLUGIN_DIR/mcp/node_modules" ]; then
  tmux display-message "tmux-claude-agents: installing dependencies..."
  (cd "$PLUGIN_DIR" && "$BUN" install --silent 2>/dev/null)
  (cd "$PLUGIN_DIR/mcp" && "$BUN" install --silent 2>/dev/null)
fi

# Wrap a string in single quotes, escaping any embedded single quotes.
# This handles spaces and special characters in $BUN and $PLUGIN_DIR.
# All variables are expanded now (at plugin load time), not at key-press time.
sq() { printf "'%s'" "${1//\'/\'\\\'\'}"; }

CMD="$(sq "$BUN") run $(sq "$PLUGIN_DIR")/cli.ts"

tmux bind-key O run-shell "$CMD menu"
