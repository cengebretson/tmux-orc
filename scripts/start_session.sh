#!/usr/bin/env bash
set -euo pipefail

PLUGIN_DIR="$(cd "$(dirname "$0")/.." && pwd)"
MCP_PORT="${CLAUDE_AGENTS_MCP_PORT:-7777}"
MCP_URL="http://localhost:${MCP_PORT}"
AGENTS_CONFIG=".claude/agents.json"
USE_CURRENT_PANE=false

# --- parse args ---

for arg in "$@"; do
  case "$arg" in
    --here) USE_CURRENT_PANE=true ;;
    --config=*) AGENTS_CONFIG="${arg#--config=}" ;;
    *) AGENTS_CONFIG="$arg" ;;
  esac
done

# --- validate ---

if [[ ! -f "$AGENTS_CONFIG" ]]; then
  echo "Error: $AGENTS_CONFIG not found. Create it with your worker definitions." >&2
  exit 1
fi

if ! command -v jq &>/dev/null; then
  echo "Error: jq is required (brew install jq)" >&2
  exit 1
fi

if ! command -v bun &>/dev/null; then
  echo "Error: bun is required (brew install bun)" >&2
  exit 1
fi

# --- start MCP server ---

"$PLUGIN_DIR/scripts/start_mcp.sh" "$MCP_PORT"
sleep 1

# --- create or reuse orchestrator pane ---

if [[ "$USE_CURRENT_PANE" == true ]]; then
  ORCH_PANE="$TMUX_PANE"
  tmux setenv MCP_URL "$MCP_URL"
  tmux setenv AGENTS_CONFIG "$AGENTS_CONFIG"
else
  ORCH_PANE=$(tmux new-window -P -F "#{pane_id}" -n "agents" \
    -e "MCP_URL=$MCP_URL" -e "AGENTS_CONFIG=$AGENTS_CONFIG")
fi

# --- send orchestrator prompt ---

ORCH_PROMPT=$(cat "$PLUGIN_DIR/templates/orchestrator.md" \
  | sed "s|{{mcp_url}}|$MCP_URL|g" \
  | sed "s|{{agents_config}}|$AGENTS_CONFIG|g")

tmux send-keys -t "$ORCH_PANE" "claude" Enter

printf '%s' "$ORCH_PROMPT" | tmux load-buffer -b "orch-prompt" -
sleep 3
tmux paste-buffer -b "orch-prompt" -t "$ORCH_PANE"
tmux send-keys -t "$ORCH_PANE" "" Enter
tmux delete-buffer -b "orch-prompt" 2>/dev/null || true

echo "Orchestrator started in pane $ORCH_PANE. MCP: $MCP_URL"
