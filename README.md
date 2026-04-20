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

A tmux plugin for running multiple Claude Code agents in parallel. An orchestrator agent coordinates workers via a local MCP server acting as a message bus, with git worktrees providing per-worker repo isolation.

```
┌──────────────────┬──────────────────┐
│  pane 1          │  pane 2          │
│  orchestrator    │  worker 1        │
│  (claude)        │  (claude)        │
│                  ├──────────────────┤
│                  │  pane 3          │
│                  │  worker 2        │
│                  │  (claude)        │
└──────────────────┴──────────────────┘
         ↕                  ↕
    MCP Server (localhost:7777)
```

Workers are **pull-based** — they call `get_task` themselves when ready, making the system self-scheduling. All inter-agent communication routes through the orchestrator (hub-and-spoke), never directly between workers.

## Requirements

- [tmux](https://github.com/tmux/tmux) 3.2+
- [Claude Code](https://claude.ai/code) CLI (`claude`)
- [Bun](https://bun.sh) (`brew install bun`)
- [jq](https://jqlang.github.io/jq/) (`brew install jq`)

## Installation

### Via TPM

Add to your `tmux.conf`:

```tmux
set -g @plugin 'yourname/tmux-claude-agents'
```

Then press `prefix+I` to install.

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
set -g @claude-agents-mcp-port 7777   # default
set -g @claude-agents-notify  true    # macOS notifications
```

## Project Setup

Create `.claude/agents.json` in your project repo:

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

Once the orchestrator is running, start a job by just telling it:

> "start job auth-login"

The orchestrator reads the job file, creates the worktree, generates tasks, and calls `load_tasks`. You can start additional jobs the same way mid-session — workers pick them up automatically.

To rerun a completed job, move it back from `done/` first.

### Validating before you start

Run `validate.sh` to check your config before starting a session:

```bash
~/.tmux/plugins/tmux-claude-agents/scripts/validate.sh
~/.tmux/plugins/tmux-claude-agents/scripts/validate.sh --job=auth-login
```

It checks:
- All workers have a role file
- All skills listed in role files (`## Skills`) exist in `.claude/skills/` or the plugin's `skills/`
- All pipeline stage roles have role files
- Job frontmatter has `pipeline:` and `domain:`, and the pipeline is defined in `agents.json`
- Job hasn't already been completed (not in `done/`)

Plugins listed in role files (`## Plugins`) produce warnings — they can't be verified from the shell, so you'll need to confirm they're enabled in Claude Code settings manually.

`start_session.sh` runs validation automatically before starting. If validation fails the session won't start.

Add to `.gitignore`:

```
.worktrees/
```

## Usage

| Keybind | Action |
|---|---|
| `prefix+M` | Start a multi-agent session (new window) |
| `prefix+Alt+M` | Start in the current pane |
| `prefix+S` | Open status menu |
| `prefix+Ctrl+M` | Clean up — kill MCP server, remove worktrees |

### Starting a session

Press `prefix+M` from inside your project directory. The plugin will:

1. Start the MCP server in the background
2. Open a new window and launch the orchestrator Claude agent
3. The orchestrator spins up worker panes and waits for instructions

Once the orchestrator is running, **just tell it what to do**:

> "start job auth-login"
> "start jobs auth-login and auth-signup in parallel"

The orchestrator reads the job file, creates the worktree, generates tasks from the pipeline stages, and calls `load_tasks`. You can start additional jobs at any point mid-session the same way — workers pick them up automatically.

To pre-load a job at cold-start, pass `--job`:

```bash
~/.tmux/plugins/tmux-claude-agents/scripts/start_session.sh --job=auth-login
```

### What the orchestrator does

1. Registers the MCP server, creates worktrees, spins up workers, and loads tasks via `load_tasks`
2. Workers register, then start pulling tasks via `get_task`
3. Monitor progress with `get_status` or `prefix+S`
4. When `all_done()` returns true, aggregate results with `get_result`

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

Press `prefix+M` from your project directory. The plugin starts the MCP server and opens a new `agents` window with the orchestrator:

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

### Step 3 — Orchestrator spins up workers

The orchestrator creates a pane per worker, writes the role file as `CLAUDE.md` into `.worktrees/auth-login`, installs skills there, and pastes each worker's bootstrap prompt:

```
┌─────────────────────┬────────────────────┐
│  orchestrator       │  bob (frontend)    │
│                     │  > claude          │
│  "Spinning up       ├────────────────────┤
│   workers..."       │  rex (review)      │
│                     │  > claude          │
│                     ├────────────────────┤
│                     │  sam (security)    │
│                     │  > claude          │
└─────────────────────┴────────────────────┘
```

### Step 4 — Workers self-bootstrap

Each worker independently runs its bootstrap loop. Because it's a pipeline session, the worktree already exists — workers just register and start pulling tasks:

```
register_worker(worker_id="bob", pane_id="%23")
# worktree .worktrees/auth-login already exists — no git worktree add needed
get_task(worker_id="bob", role="frontend")
→ task p1: "Build auth login form"
```

### Step 5 — Orchestrator loads the pipeline

```json
load_tasks([
  { "id": "p1", "role": "frontend", "description": "Build login + signup forms with JWT token handling", "job": "auth-login", "stage": "build"                                       },
  { "id": "p2", "role": "review",   "description": "Review the auth frontend changes",                   "job": "auth-login", "stage": "review",   "depends_on": ["build"]            },
  { "id": "p3", "role": "security", "description": "Audit the auth flow for vulnerabilities",            "job": "auth-login", "stage": "security", "depends_on": ["build"]            },
  { "id": "p4", "role": "git",      "description": "Open a PR merging agent/auth-login into main",       "job": "auth-login", "stage": "ship",     "depends_on": ["review", "security"] }
])
```

All tasks are loaded at once. Bob picks up `p1` immediately. Rex and Sam call `get_task` but get `NO_TASKS` — the server withholds `review` and `security` until `build` is complete. They retry every 30 seconds automatically.

### Step 6 — Build stage

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

Both stages done. Orchestrator reads their results and loads the final task with the findings baked into the description:

```
get_stage_results("auth-login", "review")   → { "rex": "LGTM, 2 minor comments..." }
get_stage_results("auth-login", "security") → { "sam": "No critical issues. CSRF token missing on signup form." }
```

```json
load_tasks([{
  "id": "p4",
  "role": "git",
  "description": "Open PR: agent/auth-login → main. Review notes: LGTM, 2 minor comments. Security: add CSRF token to signup form before merging.",
  "job": "auth-login",
  "stage": "ship"
}])
```

The `git` worker picks it up, applies the CSRF fix, runs `/pr-description`, and opens a PR from `agent/auth-login` → `main`. All the pipeline's work is already on one branch — no merging needed.

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

Press `prefix+Ctrl+M` to kill the MCP server. If any worktrees are still around (e.g. session was aborted), cleanup will force-remove them and warn about any open branches.

### Inspect anytime

While the session runs, press `prefix+S` for the status menu or query the API directly:

```bash
curl localhost:7777/status                             # all worker states
curl localhost:7777/job/auth-login                     # stage breakdown
curl localhost:7777/job/auth-login/build/results       # bob's build output
```

## Status Menu

Press `prefix+S` to open the status menu. Worker entries are populated dynamically from the current session:

```
 Claude Agents 
  Status  s    ← queue depth + all worker states
  Queue   q    ← pending tasks
  Results r    ← all submitted results
  ──────────
  Worker 2     ← result for worker 2
  Worker 3     ← result for worker 3
```

Each option opens a tmux popup with formatted JSON output.

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
| `get_status()` | Orchestrator | Queue depth + all worker states |
| `all_done()` | Orchestrator | True when queue is empty and all registered workers have submitted |
| `stage_done(job, stage)` | Orchestrator | True when all tasks in a job stage are submitted |
| `get_stage_results(job, stage)` | Orchestrator | All results from a completed job stage |
| `get_jobs_status(job?)` | Orchestrator | Stage breakdown for one job or all active jobs |
| `reset_job(job)` | Orchestrator | Clears job state so the same pipeline can rerun for a new feature |

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
2. `~/.tmux/plugins/tmux-claude-agents/skills/<skill>.md` — plugin built-ins, fallback

Built-in skills (shipped in `skills/`):

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

### Adding a custom role

Role files are looked up in this order:

1. `.claude/roles/<role>.md` — project-level, takes precedence
2. `~/.tmux/plugins/tmux-claude-agents/roles/<role>.md` — plugin built-ins, fallback

To add a project-specific role, create it alongside your `agents.json`:

```
your-project/
  .claude/
    agents.json
    roles/
      data-engineer.md   ← project-specific role
```

The file defines the worker's persona, expertise, and standards. Then reference it in `agents.json`:

```json
{ "id": "dave-the-data", "role": "data-engineer", "domain": "pipelines/" }
```

If a role is used in `agents.json` but has no matching file in either location, the session will fail to start with a clear error.

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
    state.ts                 # task queue, worker state, results (in-memory)
    state.test.ts            # unit tests (bun test)
    package.json
  scripts/
    start_session.sh         # starts MCP server, creates orchestrator pane
    start_mcp.sh             # launches bun server, guards double-start via PID
    menu.sh                  # tmux display-menu for status inspection
    cleanup.sh               # kills MCP server, removes worktrees + branches
    notify.sh                # macOS notifications (Glass = done, Basso = blocked)
  templates/
    orchestrator.md          # bootstrap prompt for the orchestrator agent
    worker.md                # bootstrap prompt for each worker agent
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
cd mcp && bun test
```
