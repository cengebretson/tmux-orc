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

if ! command -v jq &>/dev/null; then
  echo "Error: jq is required (brew install jq)" >&2
  exit 1
fi

if ! command -v bun &>/dev/null; then
  echo "Error: bun is required (brew install bun)" >&2
  exit 1
fi

validate_args="--config=$AGENTS_CONFIG"
[[ -n "$JOB_NAME" ]] && validate_args="$validate_args --job=$JOB_NAME"

if ! "$PLUGIN_DIR/scripts/validate.sh" $validate_args; then
  exit 1
fi

[[ -n "$JOB_NAME" ]] && JOB_FILE=".claude/jobs/$JOB_NAME.md" && mkdir -p .claude/jobs/done

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

# --- start job watcher (if enabled) ---

WATCH_JOBS="${CLAUDE_AGENTS_WATCH_JOBS:-false}"
if [[ "$WATCH_JOBS" == "true" ]]; then
  if [[ -d ".claude/jobs" ]]; then
    tmux split-window -d -h -t "$ORCH_PANE" \
      -e "AGENTS_CONFIG=$AGENTS_CONFIG" \
      "$PLUGIN_DIR/scripts/watch_jobs.sh $ORCH_PANE .claude/jobs"
    echo "Job watcher started (watching .claude/jobs/)"
  else
    echo "Job watcher enabled but .claude/jobs/ not found — skipping"
  fi
fi

echo "Orchestrator started in pane $ORCH_PANE. MCP: $MCP_URL${JOB_NAME:+, Job: $JOB_NAME}"
