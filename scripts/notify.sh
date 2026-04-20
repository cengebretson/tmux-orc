#!/usr/bin/env bash
# Usage: notify.sh <worker_id> <done|blocked>
WORKER_ID="${1:-?}"
STATE="${2:-done}"

if [[ "$STATE" == "blocked" ]]; then
  osascript -e "display notification \"Worker $WORKER_ID is blocked\" with title \"Claude Agent\" sound name \"Basso\""
else
  osascript -e "display notification \"Worker $WORKER_ID finished\" with title \"Claude Agent\" sound name \"Glass\""
fi
