# orc — Plan

---

## ~~1. Unify config parsing~~ ✓ Done

`internal/workflow` deleted. All workflow types and methods moved into
`internal/config`. All callers use a single `config.Load` call.

---

## ~~2. Worktree contract validation~~ ✓ Done

`state.ValidateRepos(s, root)` added. Called by `orc advance` and `orc wait`
before writing state. Checks main path existence, worktree under `worktrees/`,
non-empty branch when worktree is set, cwd under a recorded worktree. 7 tests.

---

## ~~3. Extract next-action planning from `main.go`~~ ✓ Done

`internal/runner` package created. `runner.Compute` resolves workflow, stage
config, worker, prompt, and launch args. `runNext` and `runNextAction` collapsed
into `runNext` + `printDryRun`. 6 tests added.

---

## Up next

### Banner suppression

Auto-suppress the ASCII banner when stdout is not a TTY. Add `--no-banner` flag
for scripting. Most useful when piping `orc next --json` into other tools.

**Effort:** Small.

---

### `reasoning_effort` / `service_tier` in workers

Add `reasoning_effort` and `service_tier` fields to worker frontmatter so Codex
workers can declare priority tier and reasoning depth. Render them in the launch
command when set.

**Effort:** Small.

---

## Future ideas

Lower priority — worth revisiting once the core is solid.

| Idea | Notes |
|------|-------|
| Agent session completion notification | Terminal bell, tmux alert, or webhook when an agent finishes a stage. Most useful in `--tmux` mode where sessions run unattended. Controlled via `settings.notify` in `orc.yaml`. |
| Quotes and themes in `orc.yaml` | `settings.quotes: [...]` for a custom TUI quote pool; `settings.theme: catppuccin-mocha` to swap the lipgloss palette. |
| Ticket system config | `settings.ticket_system` for machine-readable source — lets the intake stage know where to fetch ticket data without hardcoding it in the stage doc. |
