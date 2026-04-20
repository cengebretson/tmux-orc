#!/usr/bin/env bash
set -euo pipefail

PLUGIN_DIR="$(cd "$(dirname "$0")/.." && pwd)"
PORT="${1:-7777}"
PID_FILE="/tmp/claude-agents-mcp.pid"

if [[ -f "$PID_FILE" ]] && kill -0 "$(cat "$PID_FILE")" 2>/dev/null; then
  echo "MCP server already running (pid $(cat "$PID_FILE"))"
  exit 0
fi

bun run "$PLUGIN_DIR/mcp/server.ts" --port "$PORT" &
echo $! > "$PID_FILE"
echo "MCP server started on port $PORT (pid $!)"
