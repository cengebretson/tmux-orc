#!/usr/bin/env bash
PORT="${CLAUDE_AGENTS_MCP_PORT:-7777}"
BASE="http://localhost:$PORT"

show() {
  local title=$1
  local url=$2
  tmux display-popup -E -T " $title " \
    "curl -sf '$url' | jq . 2>/dev/null || echo 'MCP server not running'; read -r -p '' -n1"
}

# static entries
MENU=(
  "Status"  s "run-shell '\"$0\" show status'"
  "Queue"   q "run-shell '\"$0\" show queue'"
  "Results" r "run-shell '\"$0\" show results'"
  ""        ""  ""
)

# dynamic worker entries from /status
if workers=$(curl -sf "$BASE/status" | jq -r '.workers | keys[]' 2>/dev/null); then
  for w in $workers; do
    MENU+=("Worker $w" "" "run-shell '\"$0\" show result/$w'")
  done
fi

# handle direct call from menu item: `menu.sh show <path>`
if [[ "${1:-}" == "show" ]]; then
  path="${2:-status}"
  title=$(echo "$path" | sed 's|/| |g' | awk '{for(i=1;i<=NF;i++) $i=toupper(substr($i,1,1)) substr($i,2)}1')
  show "$title" "$BASE/$path"
  exit
fi

tmux display-menu -T " Claude Agents " "${MENU[@]}"
