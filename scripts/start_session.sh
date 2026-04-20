#!/usr/bin/env bash
set -euo pipefail

PLUGIN_DIR="$(cd "$(dirname "$0")/.." && pwd)"
MCP_PORT="${CLAUDE_AGENTS_MCP_PORT:-7777}"
MCP_URL="http://localhost:${MCP_PORT}"
AGENTS_CONFIG=".claude/agents.json"
USE_CURRENT_PANE=false
JOB_NAME=""
JOB_FILE=""

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
  echo "Error: $AGENTS_CONFIG not found. Create it with your worker and pipeline definitions." >&2
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

# --- validate job file (if specified) ---

if [[ -n "$JOB_NAME" ]]; then
  JOB_FILE=".claude/jobs/$JOB_NAME.md"
  if [[ ! -f "$JOB_FILE" ]]; then
    echo "Error: job file not found: $JOB_FILE" >&2
    if [[ -d ".claude/jobs" ]]; then
      available=$(ls .claude/jobs/*.md 2>/dev/null | xargs -I{} basename {} .md | tr '\n' ' ')
      echo "  Available jobs: ${available:-none}" >&2
    fi
    exit 1
  fi

  # extract pipeline from frontmatter
  pipeline=$(awk '/^---/{f=!f;next} f && /^pipeline:/{gsub(/^pipeline:[[:space:]]*/,""); print; exit}' "$JOB_FILE")
  if [[ -z "$pipeline" ]]; then
    echo "Error: no 'pipeline:' found in frontmatter of $JOB_FILE" >&2
    exit 1
  fi

  # validate pipeline exists in agents.json
  pipeline_exists=$(jq --arg name "$pipeline" '.pipelines // [] | map(select(.name == $name)) | length' "$AGENTS_CONFIG")
  if [[ "$pipeline_exists" -eq 0 ]]; then
    echo "Error: pipeline '$pipeline' (from $JOB_FILE) not found in $AGENTS_CONFIG" >&2
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
  [[ -n "$JOB_FILE" ]] && tmux setenv JOB_FILE "$JOB_FILE"
else
  ORCH_PANE=$(tmux new-window -P -F "#{pane_id}" -n "agents" \
    -e "MCP_URL=$MCP_URL" -e "AGENTS_CONFIG=$AGENTS_CONFIG" \
    ${JOB_FILE:+-e "JOB_FILE=$JOB_FILE"})
fi

# --- send orchestrator prompt ---

ORCH_PROMPT=$(cat "$PLUGIN_DIR/templates/orchestrator.md" \
  | sed "s|{{mcp_url}}|$MCP_URL|g" \
  | sed "s|{{agents_config}}|$AGENTS_CONFIG|g" \
  | sed "s|{{job_file}}|$JOB_FILE|g")

tmux send-keys -t "$ORCH_PANE" "claude" Enter

printf '%s' "$ORCH_PROMPT" | tmux load-buffer -b "orch-prompt" -
sleep 3
tmux paste-buffer -b "orch-prompt" -t "$ORCH_PANE"
tmux send-keys -t "$ORCH_PANE" "" Enter
tmux delete-buffer -b "orch-prompt" 2>/dev/null || true

echo "Orchestrator started in pane $ORCH_PANE. MCP: $MCP_URL${JOB_NAME:+, Job: $JOB_NAME}"
