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

## ~~4. Generic worker args map~~ ✓ Done

Replaced per-field `reasoning_effort` / `service_tier` / `thinking` with
`args: map[string]string`. Codex renders as `-c key=value`; Claude renders as
`--key value`. `engine` replaces `product`.

---

## ~~5. TUI polish~~ ✓ Done

Workflow pipeline order, hotfix fixture workflow, configurable auto-refresh
(`tui_refresh` in orc.yaml, default 60s), `r` key for manual refresh, active
stories on worker detail view, interactive stage drill-in from workflow detail
(`▶` cursor, enter opens stage file with pipeline context in title).

---

## Up next

### Agent completion notification

Terminal bell, tmux alert, or webhook when an agent finishes a stage. Most
useful in `--tmux` mode where sessions run unattended. Controlled via
`settings.notify` in `orc.yaml`.

**Effort:** Medium.

---

## Future ideas

Lower priority — worth revisiting once the core is solid.

| Idea | Notes |
|------|-------|
| Quotes and themes in `orc.yaml` | `settings.quotes: [...]` for a custom TUI quote pool; `settings.theme: catppuccin-mocha` to swap the lipgloss palette. |
| Ticket system config | `settings.ticket_system` for machine-readable source — lets the intake stage know where to fetch ticket data without hardcoding it in the stage doc. |
