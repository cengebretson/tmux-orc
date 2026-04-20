#!/usr/bin/env bash
set -euo pipefail

PID_FILE="/tmp/claude-agents-mcp.pid"

# kill MCP server
if [[ -f "$PID_FILE" ]]; then
  PID=$(cat "$PID_FILE")
  if kill -0 "$PID" 2>/dev/null; then
    kill "$PID"
    echo "MCP server stopped (pid $PID)"
  fi
  rm -f "$PID_FILE"
fi

# remove worktrees
if git rev-parse --git-dir &>/dev/null; then
  for worktree in .worktrees/worker*; do
    [[ -d "$worktree" ]] || continue
    branch=$(git -C "$worktree" rev-parse --abbrev-ref HEAD 2>/dev/null || true)
    git worktree remove --force "$worktree"
    echo "Removed worktree $worktree"
    if [[ -n "$branch" && "$branch" == agent/* ]]; then
      git branch -d "$branch" 2>/dev/null && echo "Deleted branch $branch" || true
    fi
  done
fi
