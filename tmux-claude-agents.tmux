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

# CMD is expanded at source time — no runtime variable resolution needed
CMD="$BUN run $PLUGIN_DIR/cli.ts"

# Wrap a subcommand in a popup so output (including errors) is always visible.
# The popup stays open after the command finishes so the user can read the output.
tmux bind-key M   run-shell "tmux display-popup -E -w 120 -h 30 '$CMD start; echo; read -r -p \"\" -s -n1'"
tmux bind-key M-M run-shell "tmux display-popup -E -w 120 -h 30 '$CMD start --here; echo; read -r -p \"\" -s -n1'"
tmux bind-key S   run-shell "$CMD menu"
tmux bind-key C-m run-shell "tmux display-popup -E -w 120 -h 30 '$CMD cleanup; echo; read -r -p \"\" -s -n1'"
