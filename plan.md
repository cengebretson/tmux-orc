# orc — Plan

Remaining work distilled from code review and cleanup analysis. Items are
ordered by value vs. effort. Completed work from review.md and cleanup.md
has been intentionally omitted.

---

## ~~1. Unify config parsing~~ ✓ Done

`internal/workflow` deleted. All workflow types (`WorkflowDef`, `StageDef`,
`RepairStageDef`) and methods (`Names`, `Stages`, `StageNames`, `NextStage`,
`StageConfig`, `IsRepairStage`, `RepairStage`) now live in `internal/config`.
All callers use a single `config.Load` call.

---

## 2. Worktree contract validation

**What:** `orc validate` checks that recorded worktrees exist on disk, but
there is no enforcement when agents write to `STATE.yaml`. A typo in
`repos.<name>.worktree` or a missing `repos.<name>.main` silently passes
through until `orc archive` fails.

**Fix:** Add a `state.ValidateRepos(s, root)` check that `orc advance` and
`orc wait` call before writing state. Checks:

- `repos.<name>.main` points at an existing git repo
- `repos.<name>.worktree` is under `worktrees/` in the workspace
- `repos.<name>.branch` is non-empty
- `next_action.cwd` matches the active worktree when repos are set

**Effort:** Small–medium. Mostly path checks, similar to existing health code.

---

## 3. Extract next-action planning from `main.go`

**What:** `cmd/orc/main.go` owns Cobra wiring, output rendering, workflow
resolution, prompt construction, worker selection, tmux orchestration, archive
logic, and state transitions. Core behavior is untestable without invoking
command globals. JSON and non-JSON paths duplicate logic.

**Fix:** Create `internal/runner` (or `internal/orc`) that computes:

- Resolved workflow and stage config
- Next stage name and completion instruction
- Selected worker (with resolution order)
- Full prompt string
- Launch args

Keep Cobra functions focused on flags, args, and printing.

**Effort:** Large. High payoff for testability but a real refactor — scope
carefully and do it in one focused pass.

---

## Product direction note

The highest-leverage direction is reliability over features. The three items
above — unified config, worktree validation, and testable next-action planning
— make the system more trustworthy before adding new capabilities.

Once those are solid, the next natural additions are:

- `settings.notify` — agent session completion webhook / bell
- `settings.quotes` + `settings.theme` — TUI customization from `orc.yaml`
- Ticket system config (`settings.ticket_system`) for machine-readable source
