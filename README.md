# orc

Keep feature work moving across agents, sessions, and repos — without losing context.

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

orc · workspace orchestrator
```

## What it does

Agentic workflows break down at the session boundary. An agent finishes a task,
the session ends, and the next agent starts cold — no memory of what was decided,
what was built, or what still needs fixing.

`orc` fixes this with a **feature folder**: a durable context pack that travels
with the ticket. Every stage reads what the previous one wrote and writes its own
outputs into a named subfolder. Any agent — or human — can pick up mid-flight and
know exactly where things stand without asking anyone.

**Context survives everything.** Session ends, agent switches, restarts — the
feature folder is the source of truth. `orc next <ticket>` gives any agent a
complete picture in seconds.

**Each stage has one job and clear handoffs.** Stage docs define inputs, outputs,
exit criteria, and the exact `orc mark` command to run when done. Agents don't
decide what to do next — the workspace tells them.

**Policy lives in files, not code.** `orc.yaml` declares stage order, default
workers, and advance mode. Stage docs are plain markdown. Change review criteria,
add a preflight check, swap models — edit the file and the next session picks it
up immediately.

**Right agent for each job.** A fast model for implementation, a smarter one for
review, a specialist for QA. Each worker is a markdown file. Use `--worker` to
override for a single run.

**Human-in-the-loop where it counts.** `orc mark <ticket> pause` creates explicit
review gates. Agents call it when they need a human decision. `orc next <ticket>`
continues when you're ready.

**Agent-agnostic by design.** Works with Claude, Codex, or anything that can read
a file and run a shell command. No SDK dependency, no lock-in.

## Install

Download a binary from the [releases page](https://github.com/cengebretson/orc/releases),
or install with Go:

```bash
go install github.com/cengebretson/orc/cmd/orc@latest
```

Or build from source (`make build` stamps the version from the latest git tag):

```bash
git clone git@github.com:cengebretson/orc.git
cd orc
make build
```

## Dependencies

`orc` itself has no runtime dependencies beyond Go. Two optional tools unlock
additional features:

| Tool | Purpose | Install |
|------|---------|---------|
| `tmux` | Session management — `orc work` launches and attaches agent sessions | `brew install tmux` |
| `chafa` | Character-art portraits in `orc tui` (`!` character sheet) on terminals without Kitty graphics support | `brew install chafa` |

**Pixel portraits:** on kitty and Ghostty, `orc tui` renders portraits as true
pixel images natively (Kitty graphics protocol, Unicode placeholders) — no
extra tools needed. Inside tmux, add this to your `tmux.conf` so the one-time
image transmission reaches the outer terminal:

```
set -g allow-passthrough on
```

Without it — or on other terminals — portraits fall back to chafa character
art, then to built-in ASCII art if chafa is not installed. Set
`ORC_PORTRAIT=symbols` or `ORC_PORTRAIT=kitty` to override the detection.

## Getting started

### 1. Scaffold a workspace

```bash
orc init
```

Run it and answer two questions: workspace path (default: current directory)
and whether to include sample workers. Or skip the prompts with flags:

```bash
orc init --workspace ~/my-workspace --with-sample-workers
```

### 2. Run setup

Let an agent configure the workspace for your ticketing system, source control,
and preferred agents:

```bash
cd ~/my-workspace
claude "Read SETUP.md and follow the setup instructions"
# or: codex "Read SETUP.md and follow the setup instructions"
```

The agent will ask about your ticket system (Jira, GitHub Issues, etc.), repos,
and which Claude/Codex model to use for each stage. It creates worker files and
updates `ROUTER.md` with the right ticket system retrieval instructions.

### 3. Check readiness

```bash
orc doctor
```

`orc doctor` checks workspace files plus local readiness: configured worker
engines on your `PATH`, tmux availability, and any `STATE.yaml.lock` files
that could affect ticket updates. Add `--fix` to remove provably-stale locks
(dead PID, or old without a valid PID) — live locks are never touched.

### 4. Start working on a ticket

```bash
orc work STORY-123
```

This creates `features/STORY-123/` and immediately prints the intake agent
launch command. Run it — the agent fetches the ticket, populates `TICKET.md`,
`SPEC.md`, and `PLAN.md`, and updates `STATE.yaml` to `status: pending`.

### 5. Continue work

```bash
orc next STORY-123
```

Launches the agent for the current stage. The agent works, updates `STATE.yaml`,
and exits. Run `orc next` again for the next stage. Use `--dry` to preview the
launch command without executing it.

You can also use the dashboard:

```bash
orc tui
```

## Example workflow

### Stages and workers

`features/STORY-123/` is the durable handoff between agents — each writes state when done, the next picks up from the same folder. Different stages can use different workers and models.

```mermaid
flowchart TD
    W(["orc work"])
    W --> intake["intake\nintake-agent"]
    intake -->|auto| develop["develop\nbob-the-developer"]
    develop -->|manual| CR["code-review\nzach-the-reviewer"]
    CR -->|auto| PO["pr-open\nbob-the-developer"]
    PO -->|manual| QA["qa-automation\nbrian-qa"]
    PO -.->|CI failures| PR["pr-repair\nbob-the-developer"]
    PR -.-> PO
    QA -->|auto| A(["orc archive"])

    style W fill:#313244,stroke:#a6e3a1,color:#cdd6f4
    style A fill:#313244,stroke:#a6e3a1,color:#cdd6f4
    style intake fill:#313244,stroke:#cba6f7,color:#cdd6f4
    style develop fill:#313244,stroke:#cba6f7,color:#cdd6f4
    style CR fill:#313244,stroke:#cba6f7,color:#cdd6f4
    style PO fill:#313244,stroke:#cba6f7,color:#cdd6f4
    style PR fill:#313244,stroke:#f38ba8,color:#cdd6f4
    style QA fill:#313244,stroke:#cba6f7,color:#cdd6f4
```

Workers are markdown files in `workers/`. Each stage in `orc.yaml` names a worker — mix models and agents freely. Use `--worker` to override for a single run.

`auto` — agent calls `orc mark <ticket> next`, next stage picks up immediately  
`manual` — agent calls `orc mark <ticket> pause`; a human approves before continuing

---

### Agent session loop

```mermaid
flowchart TD
    N([orc next]) -->|prints launch command| R[Agent runs]

    R --> AD[orc mark next\nstage complete]
    R --> WT[orc mark pause\nhuman needed]
    R --> DN[orc mark done\nfinal stage]

    AD -->|status: pending| N
    WT -->|human resolves\norc next| N
    DN -->|status: done| END([done])

    style N fill:#313244,stroke:#a6e3a1,color:#cdd6f4
    style R fill:#313244,stroke:#89b4fa,color:#cdd6f4
    style AD fill:#313244,stroke:#a6e3a1,color:#cdd6f4
    style WT fill:#313244,stroke:#f9e2af,color:#cdd6f4
    style DN fill:#313244,stroke:#a6e3a1,color:#cdd6f4
```

State is always written to `STATE.yaml` before the session ends — the next agent
or human picks up exactly where the last one left off.

When a session is paused (`orc mark <ticket> pause`), the reason is recorded in history and status is set to `paused`. Running `orc next <ticket>` again will show the pause reason and offer to relaunch with a recovery prompt built from the current feature context — so the agent resumes with full awareness of what was in progress and why it stopped.

---

### JIT tasks

Sometimes you need to run a one-off agent task that doesn't belong in the pipeline — a spot check, a secondary review, an exploratory investigation. `orc jit` handles this without touching the pipeline stage or status.

```bash
orc jit STORY-123 --worker zach-the-reviewer "make sure the auth middleware handles token expiry correctly"
```

The agent is launched with the same orientation prompt used by `orc next` — it reads `STATE.yaml`, `TICKET.md`, and `SPEC.md` to understand the ticket, then does the requested task. Output goes to `features/<slug>/jit/<timestamp>/`. The pipeline stage and status are unchanged throughout.

`runtime.jit` is written to `STATE.yaml` before launch so the task is visible in `orc status` and the TUI:

```
STORY-123   active   default/develop + jit   bob-developer
```

When the agent finishes, it runs:

```bash
orc mark STORY-123 jit "confirmed token expiry is handled — no issues found"
```

This appends a history entry and clears `runtime.jit`. A second `orc jit` call is blocked while one is already running — clear it first with `orc mark <ticket> jit`.

Use `--dry` to preview the full prompt and launch command without executing, and `--tmux` to send the task to the ticket's existing tmux session.

---

### Helpful plugins

These tools work well alongside `orc` and are worth setting up before you start.

#### context-mode

[context-mode](https://github.com/mksglu/context-mode) is a Claude Code plugin that keeps large tool outputs out of your context window and captures session state for resumption. It matters here because orc sessions are long — agents read `STATE.yaml`, stage docs, history, and file trees. Without context-mode, that output accumulates and pushes earlier context out of the window. With it, only summaries land in context; raw output stays in a local knowledge base the model can search on demand.

Install once, then it runs automatically in every session:

```bash
claude mcp add context-mode -- npx -y @context-mode/mcp@latest
```

Enable in settings:

```json
{
  "enabledPlugins": {
    "context-mode@context-mode": true
  }
}
```

Key commands: `/ctx-stats` to see how much context was saved, `/ctx-upgrade` to update.

---

#### GitHub MCP

The [GitHub MCP server](https://github.com/github/github-mcp-server) gives agents native access to GitHub — PRs, issues, review comments, CI run status — without shelling out to `gh`. In an orc workflow this matters most during `pr-open`, `pr-repair`, and `code-review` stages: agents can read PR state, post review comments, and check CI results directly through the MCP tool rather than constructing shell commands and parsing their output.

Install:

```bash
claude mcp add github -s user -- docker run -i --rm -e GITHUB_PERSONAL_ACCESS_TOKEN ghcr.io/github/github-mcp-server
```

Or via the Claude Desktop settings if you prefer the UI. Requires a GitHub personal access token with `repo` and `pull_requests` scopes.

Once connected, agents in your workspace will automatically use `mcp__github__*` tools when they need PR or issue context — no changes to stage docs required.

---

## Gallery

### Dashboard (`orc tui`)

![orc tui — workspace dashboard showing features, workflows, workers, and routes](docs/tui.png)

---

## Commands

### Human commands

- `orc init` — scaffold a new workspace
  - `--workspace <path>` — scaffold at a specific path
  - `--with-sample-workers` — include sample worker files
  - `--dry-run` — preview without writing
  - `--force` — overwrite existing files
- `orc doctor` — check workspace health plus `orc.yaml`, local tools, worker engines, tmux, and state locks
  - `orc doctor <ticket>` — validate a ticket's `STATE.yaml`: workflow, stage, worker, next action, repos, and worktrees
  - `--fix` — remove provably-stale state locks (dead PID or old without a valid PID); live locks are never touched
- `orc status` — show all features and their current workflow/stage
  - `orc status <ticket>` — show full details for a specific ticket
  - `--json` — output as JSON for scripting
- `orc work <ticket>` — create the feature folder for a ticket
  - `--workflow <name>` — use a named workflow instead of the configured default
  - `--tmux` — also enable a tmux session for this ticket
  - `--next` — create the feature folder and immediately launch the first stage
- `orc next <ticket>` — launch the next agent for a ticket
  - `--dry` — preview the launch command without running it
  - `--json` — next action as JSON for CI or scripting
  - `--worker <id>` — override the selected worker for one launch
- `orc jit <ticket> --worker <id> "<instruction>"` — run a one-off agent task outside the pipeline
  - `--dry` — preview the resolved worker and prompt without launching
  - `--tmux` — send to the ticket's existing tmux session instead of foreground
- `orc attach <ticket>` — attach to the tmux session for a ticket
- `orc archive <ticket>` — archive a completed feature, remove worktrees
- `orc delete <ticket>` — permanently delete a feature folder (only allowed when status is `done` or `archived`)
- `orc tui` — open the interactive dashboard

### Agent commands

These are called by agents at the end of each session. They are hidden from `orc --help` but visible via `orc help-all`.

- `orc mark <ticket> start` — begin a fresh session; allowed from `pending` or `ready`
- `orc mark <ticket> resume` — continue a paused session; allowed from `paused` only; clears the human-directed next action
- `orc mark <ticket> next` — mark the current stage complete and advance (`done` if no stages remain)
  - `--stage <name>` — jump to a specific stage (e.g. send back to develop after review)
  - `--worker <id>` — override the worker for the next stage
  - `--result "<summary>"` — record what was accomplished in history
- `orc mark <ticket> pause "<reason>"` — pause for human input, approval, or an external blocker
- `orc mark <ticket> done` — mark active, ready, or paused work as done
- `orc mark <ticket> jit "<summary>"` — record a jit task as complete and clear `runtime.jit`

`orc mark` validates transitions before writing `STATE.yaml`: pending tickets must be started before `next`, `done` is rejected from `pending`, stage and worker overrides must exist, and invalid workspace config blocks advancement.

## Workspace layout

```
my-workspace/
  AGENTS.md          shared context and routing rules (Claude + Codex)
  CLAUDE.md          imports AGENTS.md (Claude entrypoint)
  ROUTER.md          which repo owns each task, worktree paths
  TOOLS.md           approved tools, MCP servers, external systems
  RULES.md           approval, state update, and cost rules
  SETUP.md           one-time setup — run with your agent after init
  .gitignore         excludes worktrees/

  features/
    _template/       copied for each new ticket
      STATE.yaml     durable state machine for the ticket
      TICKET.md      ticket summary and acceptance criteria
      SPEC.md        context, scope, and open questions
      PLAN.md        approach and steps
      DECISIONS.md   decisions and rationale
      # stage subfolders such as develop/, code-review/, and pr-open/
      # are created by agents when those stages write outputs
    _archive/        completed features moved here by `orc archive`

  workers/
    _template.md     worker definition template
    intake-agent.md  fetches tickets, populates feature folder
    # add more workers per stage

  stages/
    intake.md        load ticket context — runs first for every ticket
    develop.md       implementation
    code-review.md   review implementation before opening PR
    pr-open.md       preflight checks, open PR, handoff for review
    pr-repair.md     fix CI failures, review feedback, conflicts
    qa-automation.md implement and run automated tests
    # plain markdown — no frontmatter; flow control lives in orc.yaml

  orc.yaml           workspace config — repos, workflows, loop stages, settings
  ORC.md             agent state contract — read at session start

  worktrees/         git worktrees for ticket branches (gitignored)
```

## Workspace files

The root files are the shared context every agent reads before starting work. Each has a distinct owner and purpose.

| File | Owner | Purpose |
|------|-------|---------|
| `AGENTS.md` | shared | Entry point for all agents — routing, session protocol, repo commands. Add team conventions at the bottom. |
| `CLAUDE.md` | orc | Imports `AGENTS.md`. Claude's entrypoint — do not edit. |
| `ORC.md` | orc | State contract — status values, `orc mark` commands, STATE.yaml rules. Do not add team conventions here. |
| `ROUTER.md` | user | Ticket system details, repo purposes, worktree layout. Fill in during setup. |
| `TOOLS.md` | user | Approved tools, MCP servers, CLI commands, external systems. Fill in during setup. |
| `RULES.md` | user | What requires human approval before agents act — PR gates, cost limits, destructive operations. |
| `SETUP.md` | orc | One-time setup guide. Run with your agent after `orc init` to configure repos, workers, and tool policy. |
| `orc.yaml` | user | Workflow config — repos, stage order, workers, loop stages, settings. |

`AGENTS.md` is the entry point — it fans out to everything else. `ORC.md` and `CLAUDE.md` are orc-managed and should not be edited directly. Everything else is yours to configure and extend.

## Feature folder

Every ticket is a self-contained context pack under `features/<slug>/`. Stages read what the previous one wrote and write their own outputs to a named subfolder — so any agent can pick up mid-flight without asking anyone.

```
features/STORY-123/
  STATE.yaml          orc-managed — status, stage, worker, history
  TICKET.md           intake writes   →  all stages read
  SPEC.md             intake writes   →  develop, code-review read
  PLAN.md             intake writes   →  develop reads
  DECISIONS.md        any stage writes → any stage reads

  develop/
    HANDOFF.md        develop writes  →  code-review, pr-open read
  code-review/
    REVIEW.md         code-review writes → develop, pr-open read
  pr-open/
    PR.md             pr-open writes  →  pr-repair, qa-automation, human read
  qa-automation/
    PLAN.md           qa-automation writes and reads across sessions
    RUNS.md
    RESULT.md
```

The stage subfolder names match the stage names in `orc.yaml` — provenance is always unambiguous. If you need to find what `develop` produced, look in `develop/`.

| File | Written by | Read by |
|------|-----------|---------|
| `STATE.yaml` | orc | orc, all agents |
| `TICKET.md` | intake | all stages |
| `SPEC.md` | intake | develop, code-review |
| `PLAN.md` | intake | develop |
| `DECISIONS.md` | any stage | any stage |
| `develop/HANDOFF.md` | develop | code-review, pr-open, qa-automation |
| `code-review/REVIEW.md` | code-review | develop, pr-open |
| `pr-open/PR.md` | pr-open | pr-repair, qa-automation, human |

## orc.yaml

`orc.yaml` is the workspace config. It declares repos, named workflows, loop
stages, and optional settings. See [docs/workflows.md](docs/workflows.md) for
the full configuration reference.

```yaml
settings:
  default_workflow: default
  auto_archive: false
  auto_tmux: false       # wrap every orc next launch in a tmux session automatically
  auto_next: false       # orc work immediately launches the first stage (same as --next)
  tui_refresh: 60        # dashboard auto-refresh interval in seconds
  theme: catppuccin-mocha

repos:
  - name: my-app
    path: ../my-app
    purpose: Application code, APIs, tests

workflows:
  default:
    stages:
      - name: intake
        worker: fred-documentor
        advance: auto
      - name: develop
        worker: bob-developer
        advance: manual
        loop:
          via: code-review
          worker: zach-reviewer
          max: 3
          on_max: pause
      - name: pr-open
        worker: bob-developer
        advance: manual
        loop:
          via: pr-repair
          worker: bob-developer
          max: 3
          on_max: pause
      - name: qa-automation
        worker: brian-qa
        advance: auto
```

`default_workflow` is used by `orc work <ticket>` when `--workflow` is omitted.
If it is not set, `orc work` returns an error. `advance: auto` tells agents to
run `orc mark <ticket> next` when a stage is complete; `advance: manual` tells agents to
run `orc mark <ticket> pause` so a human can review before continuing.

## STATE.yaml

Every ticket has one. Agents update it as work progresses. `orc` reads it to
route work to the right agent.

```yaml
schema_version: 1
ticket: STORY-123
slug: STORY-123-add-login
status: active
workflow: default

stage:
  worker: bob-developer
  name: develop

next_action:
  worker: bob-developer
  prompt: Implement the login feature per SPEC.md and PLAN.md.
  cwd: worktrees/my-app/STORY-123-add-login

runtime:
  tmux:                         # present when a tmux session is configured
    session: STORY-123-add-login

  jit:                          # present while a jit task is running, absent otherwise
    worker: zach-the-reviewer
    task: "check the auth middleware handles token expiry"
    started_at: "2026-06-01T13:45:00-05:00"

history:
  - at: "2026-05-28 09:00"
    stage: intake
    worker: fred-documentor
    result: ticket context loaded, SPEC.md and PLAN.md written
  - at: "2026-05-29 14:22"
    stage: develop
    worker: bob-developer
    result: paused — need product decision on refresh token TTL
  - at: "2026-05-30 09:10"
    stage: develop
    worker: bob-developer
    result: resumed after human clarified TTL should be 7 days
```

### Status values

| Status | Meaning | Set by |
|--------|---------|--------|
| `pending` | Session not yet started for the current stage | `orc work`, `orc mark <ticket> next` |
| `ready` | Human-set: cleared for the next session | human |
| `active` | Agent is actively working | `orc mark <ticket> start`, `orc mark <ticket> resume` |
| `paused` | Human needed — input, approval, or external blocker | `orc mark <ticket> pause` |
| `done` | All stages complete, or explicitly closed | `orc mark <ticket> next` (final stage) or `orc mark <ticket> done` |
| `archived` | Feature folder moved to `_archive/` | `orc archive` |

`runtime.tmux.session` is the source of truth for tmux operations once present.
Older tickets without that field fall back to the feature slug. `orc attach`,
`orc status`, `orc tui`, and archive cleanup all use the recorded runtime session
so custom or restored session names continue to work.

`runtime.jit` is present only while a one-off JIT task is open. Finish the task
with `orc mark <ticket> jit "<summary>"`; that records a history entry and clears
the JIT runtime block.

State writes use `STATE.yaml.lock` with atomic temp-file replacement. If an orc
process dies mid-write, the next state write can recover dead-PID locks and old
malformed locks automatically. `orc doctor` reports any lock files it finds so
you can tell whether a live process is holding state or a stale lock will be
recovered on the next write; `orc doctor --fix` removes the stale ones
immediately without waiting for a write.

## Workers

Markdown files with YAML frontmatter. The frontmatter defines who the worker is
and how to launch them. The body gives the agent behavioral guidance.

```markdown
---
id: bob-developer
name: Bob the Developer
engine: codex
model: gpt-5.5
args:
  reasoning_effort: high
  service_tier: medium
---

Implements features, opens PRs, and repairs CI failures.
```

`orc.yaml` declares the default worker per stage via `worker: <id>` in each stage entry. `orc next` looks up that worker, builds the prompt, and launches it.

**What goes into the prompt:**

Every launch gets a preamble pointing the agent at `AGENTS.md` and `ORC.md`, followed by the task prompt from `STATE.yaml`'s `next_action` field (or a generated one pointing at `features/<slug>/STATE.yaml` and `stages/<stage>.md`), and a closing instruction with the exact `orc mark` command to run when done — including whether the next advance is `auto` or `manual`.

When relaunching a paused or interrupted session, `orc next` builds a richer recovery prompt that also includes: recent history entries (what each prior stage did and who ran it), any partial output files already written to the current stage folder, and a checklist of key context files to read — `TICKET.md`, `SPEC.md`, `DECISIONS.md`, and the stage doc.

This means no agent ever starts cold. The prompt is a complete handoff: what the ticket is, where things stand, what this stage needs to produce, and exactly what command ends the session. The agent reads the files, does the work, runs the command — and the next agent gets the same treatment.

**Worker resolution order:**

1. `--worker <id>` flag on `orc next` — one-off override
2. `stage.worker` in `STATE.yaml` — set by a previous `orc mark <ticket> next --worker`
3. `worker:` for the current stage in `orc.yaml`

If no worker is found at any step, `orc next` exits with a clear error pointing to `orc.yaml`. Use `--dry` to preview the full launch command before running it.

---

## Further reading

- [Context Loss: Why Your AI Coding Agent Forgets](https://cleanaim.com/silent-wiring/problems/context-loss/) — CleanAim
- [Agent Memory vs. Context Engineering: What Persists Between Sessions](https://www.augmentcode.com/guides/agent-memory-vs-context-engineering) — Augment Code
- [Codified Context: Infrastructure for AI Agents in a Complex Codebase](https://arxiv.org/abs/2602.20478) — arXiv 2026
