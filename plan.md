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

## ~~2. Worktree contract validation~~ ✓ Done

`state.ValidateRepos(s, root)` added to `internal/state`. Called by `orc advance`
and `orc wait` before writing state. Checks main path existence, worktree under
`worktrees/`, non-empty branch when worktree is set, and cwd under a recorded
worktree when any worktrees are present. 7 tests added.

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
