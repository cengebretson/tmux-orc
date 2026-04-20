#!/usr/bin/env bash
set -euo pipefail

PLUGIN_DIR="$(cd "$(dirname "$0")/.." && pwd)"
MCP_PORT="${CLAUDE_AGENTS_MCP_PORT:-7777}"
MCP_URL="http://localhost:${MCP_PORT}"
AGENTS_CONFIG=".claude/agents.json"
USE_CURRENT_PANE=false
JOB_NAME=""

# --- parse args ---

for arg in "$@"; do
  case "$arg" in
    --here) USE_CURRENT_PANE=true ;;
    --config=*) AGENTS_CONFIG="${arg#--config=}" ;;
    --job=*) JOB_NAME="${arg#--job=}" ;;
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

# --- validate roles ---

find_role_file() {
  local role=$1
  local project_role=".claude/roles/$role.md"
  local plugin_role="$PLUGIN_DIR/roles/$role.md"
  if [[ -f "$project_role" ]]; then
    echo "$project_role"
  elif [[ -f "$plugin_role" ]]; then
    echo "$plugin_role"
  else
    echo ""
  fi
}

worker_count=$(jq '.workers | length' "$AGENTS_CONFIG")
for i in $(seq 0 $((worker_count - 1))); do
  role=$(jq -r ".workers[$i].role" "$AGENTS_CONFIG")
  if [[ -z "$(find_role_file "$role")" ]]; then
    echo "Error: no role file found for '$role'" >&2
    echo "  Looked in: .claude/roles/$role.md, $PLUGIN_DIR/roles/$role.md" >&2
    exit 1
  fi
done

# --- validate job (if specified) ---

if [[ -n "$JOB_NAME" ]]; then
  job_exists=$(jq --arg name "$JOB_NAME" '.jobs // [] | map(select(.name == $name)) | length' "$AGENTS_CONFIG")
  if [[ "$job_exists" -eq 0 ]]; then
    echo "Error: job '$JOB_NAME' not found in $AGENTS_CONFIG" >&2
    echo "  Available jobs: $(jq -r '.jobs // [] | map(.name) | join(", ")' "$AGENTS_CONFIG")" >&2
    exit 1
  fi
fi

# --- start MCP server ---

"$PLUGIN_DIR/scripts/start_mcp.sh" "$MCP_PORT"
sleep 1

# --- create or reuse orchestrator pane ---

if [[ "$USE_CURRENT_PANE" == true ]]; then
  ORCH_PANE="$TMUX_PANE"
  tmux setenv MCP_URL "$MCP_URL"
  tmux setenv AGENTS_CONFIG "$AGENTS_CONFIG"
  [[ -n "$JOB_NAME" ]] && tmux setenv JOB_NAME "$JOB_NAME"
else
  ORCH_PANE=$(tmux new-window -P -F "#{pane_id}" -n "agents" \
    -e "MCP_URL=$MCP_URL" -e "AGENTS_CONFIG=$AGENTS_CONFIG" \
    ${JOB_NAME:+-e "JOB_NAME=$JOB_NAME"})
fi

# --- send orchestrator prompt ---

ORCH_PROMPT=$(cat "$PLUGIN_DIR/templates/orchestrator.md" \
  | sed "s|{{mcp_url}}|$MCP_URL|g" \
  | sed "s|{{agents_config}}|$AGENTS_CONFIG|g" \
  | sed "s|{{job}}|$JOB_NAME|g")

tmux send-keys -t "$ORCH_PANE" "claude" Enter

printf '%s' "$ORCH_PROMPT" | tmux load-buffer -b "orch-prompt" -
sleep 3
tmux paste-buffer -b "orch-prompt" -t "$ORCH_PANE"
tmux send-keys -t "$ORCH_PANE" "" Enter
tmux delete-buffer -b "orch-prompt" 2>/dev/null || true

echo "Orchestrator started in pane $ORCH_PANE. MCP: $MCP_URL${JOB_NAME:+, Job: $JOB_NAME}"
