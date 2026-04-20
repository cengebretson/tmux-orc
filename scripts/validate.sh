#!/usr/bin/env bash
set -euo pipefail

PLUGIN_DIR="$(cd "$(dirname "$0")/.." && pwd)"
AGENTS_CONFIG=".claude/agents.json"
JOB_NAME=""
ERRORS=0
WARNINGS=0

for arg in "$@"; do
  case "$arg" in
    --config=*) AGENTS_CONFIG="${arg#--config=}" ;;
    --job=*)    JOB_NAME="${arg#--job=}" ;;
  esac
done

err()  { echo "  ✗ $1" >&2; ((ERRORS++)) || true; }
warn() { echo "  ⚠ $1"; ((WARNINGS++)) || true; }
ok()   { echo "  ✓ $1"; }

find_role_file() {
  local role=$1
  local project="$PLUGIN_DIR/.claude/roles/$role.md"
  local builtin="$PLUGIN_DIR/roles/$role.md"
  [[ -f "$project" ]] && echo "$project" || { [[ -f "$builtin" ]] && echo "$builtin" || echo ""; }
}

find_skill_file() {
  local skill=$1
  local project=".claude/skills/$skill.md"
  local builtin="$PLUGIN_DIR/skills/$skill.md"
  [[ -f "$project" ]] && echo "$project" || { [[ -f "$builtin" ]] && echo "$builtin" || echo ""; }
}

# extract lines from a named ## section until the next ##
section_lines() {
  local file=$1 section=$2
  awk "/^## ${section}/{found=1;next} /^##/{found=0} found" "$file"
}

# --- agents.json ---

echo "Checking $AGENTS_CONFIG..."

if [[ ! -f "$AGENTS_CONFIG" ]]; then
  err "agents.json not found: $AGENTS_CONFIG"
  exit 1
fi

if ! jq empty "$AGENTS_CONFIG" 2>/dev/null; then
  err "agents.json is not valid JSON"
  exit 1
fi

# --- workers ---

echo ""
echo "Workers:"
worker_count=$(jq '.workers | length' "$AGENTS_CONFIG")
if [[ "$worker_count" -eq 0 ]]; then
  err "no workers defined in agents.json"
else
  for i in $(seq 0 $((worker_count - 1))); do
    worker_id=$(jq -r ".workers[$i].id" "$AGENTS_CONFIG")
    role=$(jq -r ".workers[$i].role" "$AGENTS_CONFIG")
    role_file=$(find_role_file "$role")

    if [[ -z "$role_file" ]]; then
      err "worker '$worker_id': role file not found for '$role'"
      continue
    fi
    ok "worker '$worker_id' ($role): role file found"

    # check skills listed in ## Skills section
    while IFS= read -r line; do
      # match: - `/skill-name` ...
      if [[ "$line" =~ -[[:space:]]+\`/([a-z0-9_-]+)\` ]]; then
        skill="${BASH_REMATCH[1]}"
        if [[ -z "$(find_skill_file "$skill")" ]]; then
          err "worker '$worker_id' ($role): skill '/$skill' not found in .claude/skills/ or plugin skills/"
        else
          ok "worker '$worker_id' ($role): skill '/$skill' found"
        fi
      fi
    done < <(section_lines "$role_file" "Skills")

    # warn about plugins — can't verify from bash
    while IFS= read -r line; do
      # match: - `plugin-name` ... (no slash — distinguishes from skills)
      if [[ "$line" =~ -[[:space:]]+\`([a-z0-9_-]+)\` ]]; then
        plugin="${BASH_REMATCH[1]}"
        warn "worker '$worker_id' ($role): plugin '$plugin' listed — verify it is enabled in Claude Code settings"
      fi
    done < <(section_lines "$role_file" "Plugins")
  done
fi

# --- pipelines ---

echo ""
echo "Pipelines:"
pipeline_count=$(jq '.pipelines // [] | length' "$AGENTS_CONFIG")
if [[ "$pipeline_count" -eq 0 ]]; then
  warn "no pipelines defined in agents.json"
else
  for i in $(seq 0 $((pipeline_count - 1))); do
    pipeline_name=$(jq -r ".pipelines[$i].name" "$AGENTS_CONFIG")
    stage_count=$(jq ".pipelines[$i].stages | length" "$AGENTS_CONFIG")
    pipeline_ok=true
    for j in $(seq 0 $((stage_count - 1))); do
      stage_name=$(jq -r ".pipelines[$i].stages[$j].name" "$AGENTS_CONFIG")
      stage_role=$(jq -r ".pipelines[$i].stages[$j].role" "$AGENTS_CONFIG")
      if [[ -z "$(find_role_file "$stage_role")" ]]; then
        err "pipeline '$pipeline_name', stage '$stage_name': role '$stage_role' has no role file"
        pipeline_ok=false
      fi
    done
    $pipeline_ok && ok "pipeline '$pipeline_name': $stage_count stages"
  done
fi

# --- check for active job conflicts (if MCP server is running) ---

MCP_URL="${MCP_URL:-http://localhost:${CLAUDE_AGENTS_MCP_PORT:-7777}}"
if [[ -n "$JOB_NAME" ]] && curl -sf "$MCP_URL/jobs" -o /tmp/_ca_jobs.json 2>/dev/null; then
  if jq -e --arg job "$JOB_NAME" 'has($job)' /tmp/_ca_jobs.json >/dev/null 2>&1; then
    err "job '$JOB_NAME' is already active in the running MCP server — use reset_job to rerun it"
  fi
  rm -f /tmp/_ca_jobs.json
fi

# --- job file ---

if [[ -n "$JOB_NAME" ]]; then
  echo ""
  echo "Job: $JOB_NAME"

  if [[ -f ".claude/jobs/done/$JOB_NAME.md" ]]; then
    err "job '$JOB_NAME' already completed — move from .claude/jobs/done/ to rerun"
  else
    JOB_FILE=".claude/jobs/$JOB_NAME.md"
    if [[ ! -f "$JOB_FILE" ]]; then
      err "job file not found: $JOB_FILE"
    else
      pipeline=$(awk '/^---/{f=!f;next} f && /^pipeline:/{gsub(/^pipeline:[[:space:]]*/,""); print; exit}' "$JOB_FILE")
      domain=$(awk '/^---/{f=!f;next} f && /^domain:/{gsub(/^domain:[[:space:]]*/,""); print; exit}' "$JOB_FILE")

      if [[ -z "$pipeline" ]]; then
        err "job '$JOB_NAME': missing 'pipeline:' in frontmatter"
      else
        exists=$(jq --arg p "$pipeline" '.pipelines // [] | map(select(.name == $p)) | length' "$AGENTS_CONFIG")
        if [[ "$exists" -eq 0 ]]; then
          err "job '$JOB_NAME': pipeline '$pipeline' not defined in agents.json"
        else
          ok "job '$JOB_NAME': pipeline '$pipeline'"
        fi
      fi

      if [[ -z "$domain" ]]; then
        err "job '$JOB_NAME': missing 'domain:' in frontmatter"
      else
        ok "job '$JOB_NAME': domain '$domain'"
      fi
    fi
  fi
fi

# --- summary ---

echo ""
if [[ "$ERRORS" -gt 0 ]]; then
  echo "Validation failed — $ERRORS error(s), $WARNINGS warning(s)" >&2
  exit 1
elif [[ "$WARNINGS" -gt 0 ]]; then
  echo "Validation passed with $WARNINGS warning(s)"
else
  echo "Validation passed"
fi
