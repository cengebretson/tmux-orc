#!/usr/bin/env bash
set -euo pipefail

PLUGIN_DIR="$(cd "$(dirname "$0")/.." && pwd)"
ORCH_PANE="${1:?usage: watch_jobs.sh <orch_pane_id> [jobs_dir]}"
JOBS_DIR="${2:-.claude/jobs}"
AGENTS_CONFIG="${AGENTS_CONFIG:-.claude/agents.json}"

notify() {
  "$PLUGIN_DIR/scripts/notify.sh" "watcher" "$1" 2>/dev/null || true
}

on_new_file() {
  local path="$1"
  local file
  file="$(basename "$path")"

  # ignore non-.md files and anything inside done/
  [[ "$file" == *.md ]] || return
  [[ "$path" == */done/* ]] && return
  # ignore if already moved to done (fswatch can fire late)
  [[ -f "$path" ]] || return

  local job="${file%.md}"
  echo "[watch_jobs] detected: $job"

  if "$PLUGIN_DIR/scripts/validate.sh" --config="$AGENTS_CONFIG" --job="$job" 2>&1; then
    echo "[watch_jobs] sending 'start job $job' to orchestrator pane $ORCH_PANE"
    tmux send-keys -t "$ORCH_PANE" "start job $job" Enter
  else
    echo "[watch_jobs] validation failed for '$job' — not starting" >&2
    notify "blocked"
  fi
}

echo "[watch_jobs] watching $JOBS_DIR (orchestrator pane: $ORCH_PANE)"

if command -v fswatch &>/dev/null; then
  # fswatch fires on create/rename events only
  fswatch -0 --event Created --event Renamed --event MovedTo "$JOBS_DIR" | \
    while IFS= read -r -d '' path; do
      on_new_file "$path"
    done
else
  # polling fallback — check every 5 seconds for files newer than our last check
  echo "[watch_jobs] fswatch not found, using polling (brew install fswatch for better performance)"
  seen_files=""
  while true; do
    while IFS= read -r path; do
      if [[ "$seen_files" != *"$path"* ]]; then
        seen_files="$seen_files $path"
        on_new_file "$path"
      fi
    done < <(find "$JOBS_DIR" -maxdepth 1 -name "*.md" -newer "$JOBS_DIR" 2>/dev/null || true)
    sleep 5
    touch "$JOBS_DIR"  # update mtime reference for next iteration
  done
fi
