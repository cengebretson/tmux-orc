# Multi-Agent Claude + tmux Design

## Concept
Use tmux panes to run multiple Claude Code agents in parallel, with full visibility
and the ability to intervene. An orchestrator agent coordinates workers via an MCP
server acting as a message bus, with git worktrees providing isolation.

## Architecture

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

## Key Components

### MCP Server (message bus)
A small local Bun/TypeScript server. Exposes tools to all agents:

**Workers call:**
- `register_worker(worker_id, pane_id)` — register on startup so the orchestrator can health-check
- `get_task(worker_id, role)` — pull the next role-matched assignment
- `submit_result(worker_id, result)` — post output when done

**Orchestrator calls:**
- `load_tasks(tasks[])` — seed the task queue
- `get_result(worker_id)` — read a worker's submitted result
- `get_status()` — queue depth + all worker states
- `all_done()` — true when queue is empty and all registered workers have submitted
- `stage_done(job, stage)` / `get_stage_results(job, stage)` — pipeline sequencing
- `get_jobs_status(job?)` — stage breakdown for one job or all active jobs
- `reset_job(job)` — clear stage state to rerun a pipeline for a new feature

Workers are pull-based — they call `get_task()` themselves when ready,
making the system self-scheduling.

### Git Worktrees

One shared worktree per job, created by the orchestrator before dispatching workers:
```bash
git worktree add .worktrees/auth-login -b agent/auth-login
```
All workers in the same job share this worktree and branch. The review worker sees
the build worker's commits immediately. The git worker opens a single PR from
`agent/auth-login` → `main` — no cross-branch merging.

When the final stage is done, the orchestrator appends an `## Outcome` section (recap, branch, PR link) to the job file, moves it to `.claude/jobs/done/`, and removes the worktree. The branch stays for the open PR.

```bash
mv .claude/jobs/auth-login.md .claude/jobs/done/
git worktree remove .worktrees/auth-login
# after PR is merged: git branch -d agent/auth-login
```

### Project Config

`.claude/agents.json` defines two things:

- **`workers`** — the agent pool: id and role
- **`pipelines`** — reusable stage definitions: each stage names the role that handles it

```json
{
  "workers": [
    { "id": "bob", "role": "frontend" },
    { "id": "rex", "role": "review"   },
    { "id": "sam", "role": "security" },
    { "id": "git", "role": "git"      }
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

### Job Files

Jobs live as markdown files in `.claude/jobs/<name>.md`. YAML frontmatter specifies
the pipeline and domain; the body is the full feature spec passed as context to each stage.

```
.claude/
  agents.json
  jobs/
    auth-login.md
    auth-signup.md
```

Example — `.claude/jobs/auth-login.md`:

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
- Shows inline field-level errors on failure
- Mobile responsive

## Context
Backend JWT endpoint exists at `POST /api/auth/login` and returns `{ token, user }`.
Extend the existing `useAuth` hook at `src/shared/hooks/useAuth.ts`.
```

The orchestrator reads the job file, looks up its pipeline's stages, and generates
tasks automatically — one per stage with the markdown body as context.

### CLAUDE.md Files
- Root `CLAUDE.md` — shared conventions, project structure, coordination protocol
- Per-worktree `CLAUDE.md` — the worker's role file, copied in by the orchestrator during bootstrap

Add to `.gitignore`:
```
.worktrees/
```

## Orchestrator Startup Sequence
1. Read `.claude/agents.json` for workers and pipelines
2. Start MCP server as background process
3. Create a worktree per job being started (`git worktree add .worktrees/<job> -b agent/<job>`)
4. Spin up worker panes, copying the role file as CLAUDE.md and installing skills into the worktree
5. Read each job file, look up its pipeline's stages, generate one task per stage using the markdown body as context, and call `load_tasks`
6. Workers self-bootstrap, register, and call `get_task()` when ready

## Worker Bootstrap

The orchestrator pastes each worker's bootstrap prompt via tmux. Workers don't need
to know about job structure — they just register and loop on `get_task`:

```
You are worker bob, a frontend specialist.
Your worktree is already set up at .worktrees/auth-login — do not create a new one.
Register yourself, then call get_task and begin.
```

## Worker Lifecycle
```
receive prompt → register_worker() → get_task() → do work → submit_result() → get_task() → ...
```
When `get_task` returns `NO_TASKS`, wait 30 seconds and try again. Workers stay alive
for the full session and pick up new jobs as they are loaded.

## Tasks

Every task belongs to a `job` and `stage`. A **pipeline** is a reusable definition of
stages and roles. A **job** is a specific execution of a pipeline for a feature.

Tasks declare their dependencies via `depends_on: string[]` — stage names within the
same job that must complete before the task becomes claimable. The server enforces this:
workers get `NO_TASKS` until deps are met and retry automatically. All tasks can be
loaded upfront — no stage-by-stage loading required.

```json
{ "stage": "ship", "depends_on": ["review", "security"], ... }
```

The orchestrator polls `stage_done(job, stage)` then reads `get_stage_results(job, stage)`
to feed completed stage output as context into later stage descriptions.

For quick ad-hoc work, skip the job file and load a single-stage inline job directly:
```json
{ "id": "1", "role": "frontend", "description": "Fix login bug", "job": "fix-login", "stage": "build" }
```

Multiple jobs can run the same pipeline simultaneously — each has its own worktree,
branch, and independent stage state. To start a new job mid-session, tell the
orchestrator and it creates the worktree and calls `load_tasks` — workers pick it up automatically.

## Communication Rules (Hub-and-Spoke)

All inter-agent communication routes through the orchestrator — workers never talk
directly to each other. This prevents broadcast storms and keeps coordination predictable.

- Workers report results only via `submit_result` → orchestrator reads via `get_result`
- If worker B needs worker A's output, the orchestrator reads A's result and passes the
  relevant parts as a new task to B
- n workers means n communication channels (to orchestrator), not n² between workers

## macOS Notifications
Workers alert you when done or blocked:
```bash
osascript -e 'display notification "Worker bob finished" with title "Claude Agent" sound name "Glass"'
osascript -e 'display notification "Worker bob is blocked" with title "Claude Agent" sound name "Basso"'
```
Different sounds = know what needs attention without looking. Configure tmux to
highlight panes on bell:
```tmux
set -g bell-action any
set -g visual-bell on
```

## Packaging: tmux Plugin

Single TPM-installable plugin. MCP server ships as TypeScript source run directly by
Bun — no separate npm package or publish step.

```
tmux-claude-agents/
  tmux-claude-agents.tmux    # plugin entry point, registers keybinds
  cli.ts                     # primary CLI: validate, start, start-mcp, watch, menu, cleanup, notify
  mcp/
    server.ts                # HTTP entry point
    mcp.ts                   # MCP tool registrations
    routes.ts                # inspection GET handlers
    types.ts                 # shared TypeScript interfaces
    state.ts                 # in-memory task/worker/job state
    state.test.ts            # bun test
    package.json
  scripts/                   # bash backups (not invoked directly)
  templates/
    orchestrator.md          # bootstrap prompt for the orchestrator
    worker.md                # bootstrap prompt for each worker
  roles/                     # built-in role files (frontend, backend, review, etc.)
  skills/                    # built-in skill files (/pr-description, /security-review, etc.)
```

Install via TPM:
```tmux
set -g @plugin 'yourname/tmux-claude-agents'
```

Configure in `tmux.conf`:
```tmux
set -g @claude-agents-mcp-port   7777   # default
set -g @claude-agents-notify     true   # macOS notifications (Glass = done, Basso = blocked)
set -g @claude-agents-watch-jobs true   # auto-start jobs dropped into .claude/jobs/
```

Keybinds:
```
prefix+M        start session (new window)
prefix+Alt+M    start in current pane
prefix+S        status menu
prefix+Ctrl+M   cleanup
```

## MCP Transport: HTTP/SSE

HTTP/SSE (not stdio) so the server runs persistently in the background and serves
multiple agents simultaneously. The plugin injects `MCP_URL` into each pane's
environment at creation time — agents don't need prior MCP configuration.

## Skills and Plugins

The orchestrator copies skill files into each job's worktree at `.worktrees/<job>/.claude/commands/`
during bootstrap, making them available as slash commands. Lookup order:

1. `.claude/skills/<skill>.md` — project-level, takes precedence
2. `~/.tmux/plugins/tmux-claude-agents/skills/<skill>.md` — plugin built-ins, fallback

Workers have access to all skills — no per-worker filtering. Each role file's
`## Skills` and `## Plugins` sections document what's relevant for that role.

## Advantages Over Native Subagents
- Full visibility — watch each agent work in real time
- Can intervene, redirect, or correct mid-task
- Agents persist across long-running interactive sessions
- MCP server gives structured coordination vs file polling

## Open Questions / Next Steps
- [ ] Add tmux-logging plugin for full output capture
- [ ] Test with a small real task (e.g. auth feature with React + FastAPI)

## Known Issues / Follow-up

- [ ] **`cli.ts` tmux paths untested** — `start`, `watch`, `menu`, `cleanup` have no test coverage. The logic is non-trivial and failures will be hard to diagnose. Need an integration test strategy or at minimum manual test runbook.

- [ ] **Knowledge file path is cwd-relative** — `state.ts` writes `.claude/knowledge/<role>.md` relative to wherever the MCP server process started. If launched from the wrong directory the files go to the wrong place or fail silently. Should either assert cwd on startup or make the path configurable.

- [ ] **Knowledge file loading is unenforced** — `orchestrator.md` instructs the LLM to read knowledge files when building task descriptions, but there is no hard enforcement. If the LLM skips it, past resolutions are ignored. Consider injecting knowledge file content server-side into the task description returned by `get_task`.

- [ ] **`resolveBlock` silently skips file write if `currentTask` is missing** — if a worker's task reference is lost (e.g. after a `reset`) the resolution is recorded in worker state but never written to the knowledge file. Should log a warning or return an error.

- [ ] **No state persistence** — MCP server state is fully in-memory. A crash or restart mid-job loses all task assignments, worker states, and stage results. For long-running jobs this is a real failure mode. Options: periodic state snapshot to disk, or replay from a task log.
