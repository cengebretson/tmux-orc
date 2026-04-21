# tmux-claude-agents

```
⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢀⡀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢠⣿⣿⡄⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠀⠀⠀⣀⣤⣶⣧⣄⣉⣉⣠⣼⣶⣤⣀⠀⠀⠀⠀⠀⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠀⢰⣿⣿⣿⣿⡿⣿⣿⣿⣿⢿⣿⣿⣿⣿⡆⠀⠀⠀⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠀⣼⣤⣤⣈⠙⠳⢄⣉⣋⡡⠞⠋⣁⣤⣤⣧⠀⠀⠀⠀⠀⠀⠀
⠀⢲⣶⣤⣄⡀⢀⣿⣄⠙⠿⣿⣦⣤⡿⢿⣤⣴⣿⠿⠋⣠⣿⠀⢀⣠⣤⣶⡖⠀
⠀⠀⠙⣿⠛⠇⢸⣿⣿⡟⠀⡄⢉⠉⢀⡀⠉⡉⢠⠀⢻⣿⣿⡇⠸⠛⣿⠋⠀⠀
⠀⠀⠀⠘⣷⠀⢸⡏⠻⣿⣤⣤⠂⣠⣿⣿⣄⠑⣤⣤⣿⠟⢹⡇⠀⣾⠃⠀⠀⠀
⠀⠀⠀⠀⠘⠀⢸⣿⡀⢀⠙⠻⢦⣌⣉⣉⣡⡴⠟⠋⡀⢀⣿⡇⠀⠃⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⢸⣿⣧⠈⠛⠂⠀⠉⠛⠛⠉⠀⠐⠛⠁⣼⣿⡇⠀⠀⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠸⣏⠀⣤⡶⠖⠛⠋⠉⠉⠙⠛⠲⢶⣤⠀⣹⠇⠀⠀⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠀⠀⠀⢹⣿⣶⣿⣿⣿⣿⣿⣿⣶⣿⡏⠀⠀⠀⠀⠀⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠀⠀⠀⠈⠉⠉⠉⠛⠛⠛⠛⠉⠉⠉⠁⠀⠀⠀⠀⠀⠀⠀⠀⠀
```

A tmux plugin for running multiple Claude Code agents in parallel. An orchestrator agent coordinates workers via a local MCP server acting as a message bus, with git worktrees providing per-job repo isolation.

```
┌─────────────────────────────────┐   ┌──────────────────────────────────┐
│  agents window                  │   │  auth-login window               │
│  orchestrator (claude)          │   │  worker-bob     │  worker-rex    │
│                                 │   │  (claude)       │  (claude)      │
└─────────────────────────────────┘   └──────────────────────────────────┘
         ↕                                       ↕
    MCP Server (localhost:7777)
```

Workers are **pull-based** — they call `get_task` themselves when ready, making the system self-scheduling. All inter-agent communication routes through the orchestrator (hub-and-spoke), never directly between workers.

## Quickstart

```bash
# 1. Install (see Installation below), then from inside your project:
# 1. Edit .claude/agents.json to match your project, then write a job file:
#    .claude/jobs/my-feature.md

# 3. Validate your setup:
bun run ~/.tmux/plugins/tmux-claude-agents/cli.ts validate --job=my-feature

# 4. Start a session — workers spawn automatically when one job file exists:
prefix+O  # opens the control menu → Start session

# 5. Watch the orchestrator load tasks and workers pick them up.
#    Press prefix+O at any time to check status, inspect results, or clean up.
```

## Requirements

- [tmux](https://github.com/tmux/tmux) 3.2+
- [Claude Code](https://claude.ai/code) CLI (`claude`)
- [Bun](https://bun.sh) (`brew install bun`)

## Installation

### Via TPM

Add to your `tmux.conf`:

```tmux
set -g @plugin 'yourname/tmux-claude-agents'
```

Press `prefix+I` (TPM install). Dependencies are installed automatically on first use.

### Manual

```bash
git clone https://github.com/yourname/tmux-claude-agents ~/.tmux/plugins/tmux-claude-agents
~/.tmux/plugins/tmux-claude-agents/tmux-claude-agents.tmux
```

Install MCP server dependencies:

```bash
cd ~/.tmux/plugins/tmux-claude-agents/mcp && bun install
```

### Configuration

Add to `tmux.conf`:

```tmux
set -g @claude-agents-bun-path   /opt/homebrew/bin/bun   # default; override if bun is elsewhere
set -g @claude-agents-mcp-port   7777                    # default
set -g @claude-agents-notify     true                    # macOS notifications (default: true)
set -g @claude-agents-watch-jobs true                    # auto-start jobs dropped into .claude/jobs/
set -g @claude-agents-layout     windows                 # windows | sessions | panes (default: windows)
```

On macOS, system notifications fire automatically (set `@claude-agents-notify false` to disable). Two distinct sounds:
- **Glass** — worker finished (triggered via `notify` CLI command)
- **Basso** — worker is blocked and needs intervention

## Project Setup

Press `prefix+O` from inside your project. If `.claude/` hasn't been initialized yet, the menu shows **Init project** automatically. Or run it directly:

```bash
bun run ~/.tmux/plugins/tmux-claude-agents/cli.ts init
```

This creates `.claude/agents.json`, a sample job file, empty `roles/` and `skills/` override directories, and adds `.mcp.json` and `.worktrees/` to `.gitignore` automatically.

### Manual setup

Alternatively, create `.claude/agents.json` by hand:

```json
{
  "workers": [
    { "id": "bob",  "role": "frontend" },
    { "id": "rex",  "role": "review"   },
    { "id": "sam",  "role": "security" },
    { "id": "git",  "role": "git"      }
  ],
  "pipelines": [
    {
      "name": "frontend",
      "stages": [
        { "name": "build",    "role": "frontend" },
        { "name": "review",   "role": "review"   },
        { "name": "security", "role": "security" },
        { "name": "ship",     "role": "git"      }
      ]
    }
  ]
}
```

`agents.json` defines your permanent workforce and pipeline definitions. Jobs are separate markdown files.

### Job files

Each job lives in `.claude/jobs/<name>.md`. The frontmatter specifies which pipeline to use and the domain to work in. The body is the full spec — the orchestrator reads it to generate task descriptions for each stage.

When a job completes, the orchestrator appends an `## Outcome` section and moves the file to `.claude/jobs/done/`. Pending jobs are what's in `.claude/jobs/`; completed jobs are in `done/`.

```
.claude/
  agents.json
  jobs/
    auth-signup.md      ← pending
    done/
      auth-login.md     ← completed (has ## Outcome section)
  roles/        ← optional project-specific role overrides
  skills/       ← optional project-specific skill overrides
```

Example `.claude/jobs/auth-login.md` (before completion):

```markdown
---
pipeline: frontend
domain: src/frontend/auth/login/
---

## Goal
Build the login flow with JWT token handling.

## Acceptance criteria
- Email + password form with client-side validation
- JWT stored in httpOnly cookie, not localStorage
- Redirects to /dashboard on success
- Shows inline errors on failure

## Context
Backend JWT endpoint already exists at `POST /api/auth/login`.
Extend `src/shared/hooks/useAuth.ts`, don't replace it.

## Related
- Linear: AUTH-42
```

After completion, the same file in `done/` has an outcome appended:

```markdown
## Outcome

**Completed:** 2026-04-20
**Branch:** agent/auth-login
**PR:** https://github.com/org/repo/pull/124

### Recap
Login form built with JWT stored in httpOnly cookie. useAuth hook extended.
Review: LGTM with 2 minor comments addressed. Security: CSRF token added to
form after sam flagged it. PR opened from agent/auth-login → main.
```

### Auto-starting jobs (job watcher)

With `@claude-agents-watch-jobs true` in your `tmux.conf`, a watcher runs alongside the orchestrator. Drop any `.md` file into `.claude/jobs/` and it will be picked up automatically — validated, then sent to the orchestrator as a `start job <name>` prompt.

```bash
cp my-feature.md .claude/jobs/   # orchestrator starts it automatically
```

Invalid job files (failed validation) are skipped with a notification rather than starting a broken session.

Once the orchestrator is running, start a job by just telling it:

> "start job auth-login"

The orchestrator reads the job file, creates the worktree, generates tasks, and calls `load_tasks`. You can start additional jobs the same way mid-session — workers pick them up automatically.

To rerun a completed job, move it back from `done/` first.

### Validating before you start

Run `cli.ts validate` to check your config before starting a session:

```bash
bun run ~/.tmux/plugins/tmux-claude-agents/cli.ts validate
bun run ~/.tmux/plugins/tmux-claude-agents/cli.ts validate --job=auth-login
```

It checks:
- All workers have a role file
- All skills listed in role files (`## Skills`) exist in `.claude/skills/` or the plugin's `skills/`
- All pipeline stage roles have role files
- Job frontmatter has `pipeline:` and `domain:`, and the pipeline is defined in `agents.json`
- Job hasn't already been completed (not in `done/`)

Plugins listed in role files (`## Plugins`) produce warnings — they can't be verified from the shell, so you'll need to confirm they're enabled in Claude Code settings manually.

`cli.ts start` runs validation automatically before starting. If validation fails the session won't start.

## Usage

| Keybind | Action |
|---|---|
| `prefix+O` | Open control panel (init / start / status / jobs / cleanup) |

### Starting a session

Press `prefix+O` from inside your project directory and select **Start session**. The plugin will:

1. Start the MCP server in the background
2. Open a new `agents` window and launch the orchestrator Claude agent
3. Spawn workers according to `@claude-agents-layout` (default: one window per job, workers as panes inside it)

Once the orchestrator is running, **just tell it what to do**:

> "start job auth-login"
> "start jobs auth-login and auth-signup in parallel"

The orchestrator reads the job file, creates the worktree, generates tasks from the pipeline stages, and calls `load_tasks`. You can also start a pending job via `prefix+O` → **Start \<job-name\>** — it appears automatically for any job file not yet loaded into the running session.

To pre-load a job at cold-start, pass `--job`:

```bash
bun run ~/.tmux/plugins/tmux-claude-agents/cli.ts start --job=auth-login
```

### What the orchestrator does

1. Registers the MCP server, creates worktrees, spins up workers, and loads all tasks via `load_tasks`
2. Workers register, then start pulling tasks via `get_task` — the server withholds tasks until their `depends_on` stages complete
3. Monitor progress with `get_status` or `prefix+O` → Status
4. Read stage output with `get_stage_results(job, stage)` as each stage completes

## Example: Auth feature pipeline

This walkthrough shows a full pipeline session: four workers, four stages, results feeding forward.

### Project config

`.claude/agents.json` (workers + pipeline definition) and `.claude/jobs/auth-login.md` (the feature spec, as shown in [Job files](#job-files)):

```json
{
  "workers": [
    { "id": "bob",  "role": "frontend" },
    { "id": "rex",  "role": "review"   },
    { "id": "sam",  "role": "security" },
    { "id": "git",  "role": "git"      }
  ],
  "pipelines": [
    {
      "name": "frontend",
      "stages": [
        { "name": "build",    "role": "frontend" },
        { "name": "review",   "role": "review"   },
        { "name": "security", "role": "security" },
        { "name": "ship",     "role": "git"      }
      ]
    }
  ]
}
```

### Step 1 — Start the session

Press `prefix+O` → **Start session** from your project directory. The plugin starts the MCP server and opens a new `agents` window with the orchestrator:

```
┌──────────────────────────────────────────┐
│  agents window                           │
│                                          │
│  > claude                                │
│  [orchestrator reading prompt...]        │
│                                          │
└──────────────────────────────────────────┘
```

### Step 2 — Orchestrator creates the pipeline worktree

Before spinning up any workers, the orchestrator creates a single shared worktree for the job:

```bash
git worktree add .worktrees/auth-login -b agent/auth-login
```

All pipeline workers (`bob`, `rex`, `sam`, `git`) will work inside `.worktrees/auth-login` on branch `agent/auth-login`. This means the review worker sees bob's commits immediately, and the git worker opens one PR from `agent/auth-login` → `main`.

### Step 3 — CLI spawns workers

The CLI spawns workers according to `@claude-agents-layout`. With the default `windows` layout, each job gets a dedicated tmux window with worker panes split inside it. The orchestrator stays in the `agents` window:

```
┌─────────────────────────────────┐
│  agents window                  │
│  orchestrator                   │
│  > claude                       │
└─────────────────────────────────┘

┌─────────────────────────────────┐
│  auth-login window              │
│  bob (frontend)  │ rex (review) │
│  > claude        │ > claude     │
│                  ├──────────────┤
│                  │ sam (security│
│                  │ > claude     │
└─────────────────────────────────┘
```

### Step 4 — Orchestrator loads all tasks

All four tasks are loaded upfront with `depends_on` declaring the stage ordering:

```json
load_tasks([
  { "id": "p1", "role": "frontend", "description": "Build login + signup forms with JWT token handling", "job": "auth-login", "stage": "build"                                        },
  { "id": "p2", "role": "review",   "description": "Review the auth frontend changes",                   "job": "auth-login", "stage": "review",   "depends_on": ["build"]             },
  { "id": "p3", "role": "security", "description": "Audit the auth flow for vulnerabilities",            "job": "auth-login", "stage": "security", "depends_on": ["build"]             },
  { "id": "p4", "role": "git",      "description": "Open PR from agent/auth-login → main",               "job": "auth-login", "stage": "ship",     "depends_on": ["review", "security"] }
])
```

The server withholds `review`, `security`, and `ship` until their dependencies complete. No manual stage-by-stage loading needed.

### Step 5 — Workers self-bootstrap

Each worker registers and starts calling `get_task`. The worktree already exists — no `git worktree add` needed:

```
register_worker(worker_id="bob", pane_id="%23")
get_task(worker_id="bob", role="frontend")
→ p1: "Build login + signup forms..."   ✓ (no depends_on — available immediately)

get_task(worker_id="rex", role="review")
→ NO_TASKS  (depends_on: ["build"] — withheld until build completes)
```

Rex and Sam retry every 30 seconds automatically.

### Step 6 — Build stage completes

Bob works in `.worktrees/auth-login`. The orchestrator polls:

```
stage_done(job="auth-login", stage="build") → false ... false ... true ✓
```

Bob submits his result:

```
submit_result(worker_id="bob", result="Login + signup forms complete. JWT stored in httpOnly cookie. Files: src/frontend/auth/Login.tsx, Signup.tsx, useAuth.ts")
```

### Step 7 — Review + security run in parallel

Both `review` and `security` depend only on `build`, so they run simultaneously. Rex and Sam both call `get_task` and pick up their tasks. The orchestrator polls both:

```
stage_done("auth-login", "review")   → false ... true ✓
stage_done("auth-login", "security") → false ... false ... true ✓
```

```
┌─────────────────────┬────────────────────┐
│  orchestrator       │  bob (frontend)    │
│                     │  ✓ submitted       │
│  "build done,       ├────────────────────┤
│   review+security   │  rex (review)      │
│   running..."       │  [reading diff...] │
│                     ├────────────────────┤
│                     │  sam (security)    │
│                     │  [auditing auth..] │
└─────────────────────┴────────────────────┘
```

### Step 8 — Ship stage

Both stages done. The server automatically releases `p4` — its `depends_on: ["review", "security"]` is now satisfied. Git picks it up on its next `get_task` retry:

```
get_task(worker_id="git", role="git")
→ p4: "Open PR from agent/auth-login → main"   ✓
```

The orchestrator can read the findings for its own monitoring:

```
get_stage_results("auth-login", "review")   → { "rex": "LGTM, 2 minor comments..." }
get_stage_results("auth-login", "security") → { "sam": "No critical issues. CSRF token missing on signup form." }
```

The git worker reads the diff, runs `/pr-description`, and opens a PR from `agent/auth-login` → `main`. All the pipeline's work is already on one branch — no merging needed.

### Step 9 — Done

```
stage_done("auth-login", "ship") → true
```

macOS notification fires: **"Worker git finished"** (Glass sound).

The orchestrator appends an `## Outcome` section to the job file, archives it, then removes the worktree:

```bash
# append outcome to job file
cat >> .claude/jobs/auth-login.md << 'EOF'

## Outcome

**Completed:** 2026-04-20
**Branch:** agent/auth-login
**PR:** https://github.com/org/repo/pull/124

### Recap
Login form built with JWT stored in httpOnly cookie. useAuth hook extended.
Review: LGTM, 2 minor comments addressed. Security: CSRF token added after
sam flagged it. PR opened from agent/auth-login → main.
EOF

# archive and clean up
mv .claude/jobs/auth-login.md .claude/jobs/done/
git worktree remove .worktrees/auth-login
# after PR is merged: git branch -d agent/auth-login
```

### Inspect and clean up

Press `prefix+O` to open the control panel. Select **Cleanup…** (`x`) to kill the MCP server and remove worktrees — a confirmation prompt prevents accidental runs. Query the API directly too:

```bash
curl localhost:7777/status                             # all worker states
curl localhost:7777/job/auth-login                     # stage breakdown
curl localhost:7777/job/auth-login/build/results       # bob's build output
```

## Control Panel

Press `prefix+O` to open the context-aware control panel. Items shown depend on current state:

**No session running:**
```
 Claude Orc 
  Start session    ← start MCP + orchestrator
  Start here       ← start in current pane
  ──────────
  New job…         ← wizard to create a job file
  Validate         ← pre-flight config check
```

**Session running:**
```
 Claude Orc 
  Status           ← queue depth + all worker states
  Queue            ← pending tasks
  Results          ← all submitted results
  Jobs             ← per-job stage breakdown
  ──────────
  New job…         ← create and optionally start a new job
  ──────────
  Start auth-login ← start a pending job (one entry per unloaded job file)
  ──────────
  Worker bob       ← result for worker bob
  Worker rex       ← result for worker rex
  ──────────
  Cleanup…         ← kill MCP server + remove worktrees (with confirmation)
```

Pending job entries (`Start <name>`) appear automatically — the menu cross-references active jobs in the MCP server against `.claude/jobs/*.md` and shows only unloaded ones. Each option opens a tmux popup with formatted JSON output.

## MCP Server

The MCP server runs as a local HTTP/SSE server (Bun/TypeScript) bundled inside the plugin — no separate install needed.

### MCP Tools (used by agents)

| Tool | Called by | Description |
|---|---|---|
| `register_worker(worker_id, pane_id)` | Worker | Registers pane ID for health checking |
| `load_tasks(tasks[])` | Orchestrator | Seeds the task queue |
| `get_task(worker_id, role)` | Worker | Pulls next role-matched task |
| `submit_result(worker_id, result)` | Worker | Posts completed output |
| `get_result(worker_id)` | Orchestrator | Reads a worker's result |
| `get_status()` | Orchestrator | Queue depth + all worker states (idle / working / submitted / blocked) |
| `all_done()` | Orchestrator | True when queue is empty and all registered workers are submitted or idle |
| `stage_done(job, stage)` | Orchestrator | True when all tasks in a job stage are submitted |
| `get_stage_results(job, stage)` | Orchestrator | All results from a completed job stage |
| `get_jobs_status(job?)` | Orchestrator | Stage breakdown for one job or all active jobs |
| `reset_job(job)` | Orchestrator | Clears job state so the same pipeline can rerun for a new feature |
| `get_hung_workers(threshold_ms)` | Orchestrator | Workers in working state with no activity for longer than threshold |
| `report_blocked(worker_id, reason)` | Worker | Signals stuck state — fires a Basso notification |
| `resolve_block(worker_id, resolution)` | Worker | Unblocks worker and appends resolution to `## Lessons Learned` in the role file |

### Tasks

Tasks are structured objects:

```json
{
  "id": "auth-1",
  "role": "frontend",
  "description": "Build login form",
  "job": "auth-login",
  "stage": "review",
  "depends_on": ["build"]
}
```

`depends_on` is optional — omit it for tasks with no dependencies. The server withholds a task until all listed stages are complete for that job.

Built-in roles: `backend`, `frontend`, `review`, `document`, `security`, `git`

Workers only receive tasks matching their role.

#### Tasks

Every task requires a `job` and `stage`. For multi-stage jobs the orchestrator generates tasks from the job file — one per pipeline stage, results feeding forward:

```json
[
  { "id": "p1", "role": "frontend", "description": "Build login form", "job": "auth-login", "stage": "build"                                      },
  { "id": "p2", "role": "review",   "description": "Review auth PR",   "job": "auth-login", "stage": "review",   "depends_on": ["build"]            },
  { "id": "p3", "role": "security", "description": "Security audit",   "job": "auth-login", "stage": "security", "depends_on": ["build"]            },
  { "id": "p4", "role": "git",      "description": "Open PR",          "job": "auth-login", "stage": "ship",     "depends_on": ["review", "security"] }
]
```

All tasks are loaded upfront. `depends_on` lists stage names that must complete before a task becomes claimable — the server withholds the task until all dependencies are met. Workers just get `NO_TASKS` while waiting and retry automatically.

Orchestrator polls `stage_done(job, stage)` then reads `get_stage_results(job, stage)` to feed results into later stage descriptions.

For quick ad-hoc work, use a single-stage inline job — no job file needed:

```json
[
  { "id": "1", "role": "frontend", "description": "Fix login bug",  "job": "fix-login", "stage": "build" },
  { "id": "2", "role": "backend",  "description": "Fix auth token", "job": "fix-auth",  "stage": "build" }
]
```

Two jobs can run the same pipeline simultaneously — each gets its own worktree and independent stage state.

### Skills and plugins

Workers have access to all skills (`.claude/commands/`) and MCP plugins configured for the project — there is no per-worker filtering. Each role file documents which skills and plugins that role should use via `## Skills` and `## Plugins` sections. Workers read this from their CLAUDE.md and know what tools are relevant to them.

During bootstrap the orchestrator copies skill files into each worker's worktree at `.worktrees/<id>/.claude/commands/`, making them available as slash commands. The lookup order mirrors roles:

1. `.claude/skills/<skill>.md` — project-level, takes precedence
2. `~/.tmux/plugins/tmux-claude-agents/assets/skills/<skill>.md` — plugin built-ins, fallback

Built-in skills (shipped in `assets/skills/`):

| Skill | Description |
|---|---|
| `/component-review` | Self-review a React component before submitting |
| `/accessibility-check` | Check UI for keyboard, screen reader, and contrast issues |
| `/api-review` | Review an API endpoint for validation, auth, and error handling |
| `/test-coverage` | Assess test coverage by reading source and test files |
| `/security-review` | Security-focused pass covering injection, auth, secrets, and config |
| `/doc-review` | Review documentation for accuracy, clarity, and completeness |
| `/dependency-audit` | Audit dependencies for known vulnerabilities and abandoned packages |
| `/pr-description` | Generate a structured PR description from the current branch diff |

### Custom roles and skills

Roles and skills follow the same two-level lookup — project-level takes precedence over plugin built-ins:

| | Project-level | Plugin built-in |
|---|---|---|
| Roles | `.claude/roles/<role>.md` | `~/.tmux/plugins/tmux-claude-agents/assets/roles/<role>.md` |
| Skills | `.claude/skills/<skill>.md` | `~/.tmux/plugins/tmux-claude-agents/assets/skills/<skill>.md` |

You never need to touch the plugin folder. Put files in `.claude/` and they automatically override or extend what ships with the plugin.

**Adding a new role** — create `.claude/roles/data-engineer.md` and reference it in `agents.json`:

```
.claude/
  agents.json
  roles/
    data-engineer.md   ← picked up automatically
```

```json
{ "id": "dave", "role": "data-engineer" }
```

**Overriding a built-in role** — create `.claude/roles/frontend.md` with your project's conventions. Workers in this project get your version; the plugin built-in is ignored.

**Adding a custom skill** — create `.claude/skills/my-check.md` and list it in a role file:

```markdown
## Skills
- `/my-check` — description of what it does
```

**Overriding a built-in skill** — create `.claude/skills/pr-description.md` to replace the plugin's default with a project-specific version.

`cli.ts validate` checks all roles and skills referenced in role files before the session starts, so missing files are caught early.

### Inspection Endpoints

The MCP server also exposes plain HTTP endpoints for quick inspection:

```bash
curl http://localhost:7777/status                            # queue depth + worker states
curl http://localhost:7777/queue                             # pending tasks
curl http://localhost:7777/results                           # all submitted results
curl http://localhost:7777/result/bob                        # result for a specific worker
curl http://localhost:7777/jobs                              # all job statuses
curl http://localhost:7777/job/auth-login                    # stage breakdown for one job
curl http://localhost:7777/job/auth-login/build/results      # results from a stage
```

## Architecture

```
tmux-claude-agents/
  tmux-claude-agents.tmux    # plugin entry point, registers keybinds
  mcp/
    server.ts                # HTTP entry point, routes MCP + inspection traffic
    mcp.ts                   # MCP server instance + tool registrations
    routes.ts                # inspection GET handlers
    types.ts                 # shared TypeScript interfaces
    state.ts                 # task queue, worker state, results (in-memory)
    state.test.ts            # unit tests (bun test)
    package.json
  cli.ts                     # primary CLI: init, validate, launch, new-job, start, start-mcp, watch, menu, cleanup, notify
  assets/
    templates/
      orchestrator.md        # bootstrap prompt for the orchestrator agent
      worker.md              # bootstrap prompt for each worker agent
    roles/                   # built-in role files
    skills/                  # built-in skill files
```

### Worker isolation

Worktree ownership depends on the task mode:

**Pipeline** — all workers in a pipeline share one worktree and branch, created by the orchestrator before dispatching:

```bash
git worktree add .worktrees/auth -b agent/auth
```

Sharing one branch means the review worker immediately sees the build worker's commits, and the git worker opens a single PR (`agent/auth` → `main`) with all the work on one branch.

**Standalone** — each worker gets its own worktree and branch, also created by the orchestrator:

```bash
git worktree add .worktrees/bob -b agent/bob
```

### Communication rules

- Workers report only to the orchestrator via `submit_result`, never to each other
- If worker B needs worker A's output, the orchestrator reads A's result and passes the relevant parts as a new task to B

## Running tests

```bash
bun test
```
