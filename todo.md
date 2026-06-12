# orc — TODO

Concrete cleanup and hardening items plus lower-priority future ideas.
Specced, up-next roadmap work stays in `plan.md`. Open work is at the top;
the completed findings from the 2026-06-10 architecture review are kept
below as a record.

## Open

### TUI follow-ups (soft)

- [ ] (optional) View composition (`viewDashboard`/`viewDetail`) remains
      untested; golden-file snapshot tests are the cheapest way in if it ever
      matters.

## Future ideas

Lower priority — worth revisiting once the core is solid.

- **Workspace packs** — share workers/workflows across a team. `orc pack
  push/pull/diff` — copy workers, stages, RULES.md to/from a git repo;
  two-layer model with `overrides/` for local customization.
- **Stage timing → `orc report`** — `HistoryEntry` already timestamps every
  transition (`at`); derive time-in-stage from consecutive entries and surface
  per-ticket durations via `orc report` or a `status` column. Feeds the
  evidence-collection goal, and later per-worker cost attribution.
- **TUI event feed** — once `notify.Fire` lands (plan.md), also append events
  to a workspace `events.log`; the TUI tails it into a recent-activity pane.
  Cheap after notifications ship; pointless before.

## Considered and rejected

- `internal/stage` at 0% coverage — 36 trivial lines, not worth tests.
- `internal/tmux` at 23% — wraps the tmux binary; mocking it buys little.
- `cmd/orc/main.go` at 1,388 lines — it's flag definitions and printing, which is
  exactly what the "CLI as the boundary" principle prescribes. No action.
- Cobra package-level flag vars (`markStage` etc.) — standard cobra idiom.

---

## Done — 2026-06-10 review findings

Snapshot at review time: ~10.6K lines of Go across 18 packages; 14 of 18 at
60–89% test coverage. Design principles in CLAUDE.md verified against the
code: `cmd/orc` handlers are thin wrappers over
`internal/orchestrator`/`internal/state`, state mutations are atomic
(lock → mutate → temp file → rename), error wrapping is consistently `%w`,
zero TODO/FIXME debt.

### TUI test coverage and structure (gate met — done 2026-06-11)

The TUI was the highest-churn, lowest-coverage code: `internal/tui/tui.go` was
2,151 lines (~20% of all Go in the repo) and the package sat at **6.9% coverage**
vs 60–89% everywhere else. Four of the last five commits touched the TUI; the
portrait pixel regression (fixed in a90785f) is the class of bug coverage here
would catch. The gate was "do this before the rest of the TUI roadmap in
plan.md" — coverage is now 54.1% with the file split done, so the roadmap is
unblocked. Two soft follow-ups remain under Open above.

- [x] Split `tui.go` (2,151 lines) into six per-concern files: `model.go`,
      `update.go`, `data.go`, `render.go`, `view_dashboard.go`, `view_detail.go`.
      Pure file moves — declaration list verified identical before/after.
- [x] Added `render_test.go` covering `renderRouteChain` (incl. wrapping and loop
      annotations), `renderTable`, `renderWorkflowDetail`, `renderWorkerFile`,
      `visibleFeatures`, and the primitives (`padRight`, `truncate`, `wrapText`,
      `drawBoxLabeled` width invariant). Package coverage: 6.9% → 28.3%.
- [x] Added `update_test.go` driving `handleKey`/`Update` directly with
      `tea.KeyMsg` values (no bubbletea harness needed): quit keys, cursor
      clamping, archive toggle, search mode enter/filter/clear, tab section
      cycling, collapse-returns-focus, detail open/close, worker file viewer →
      character sheet → back, workflow drill-in cursor, tmux attach guard,
      rainbow easter egg, dataMsg cursor clamp, window resize.
      Package coverage: 28.3% → 41.6%.
- [x] Coverage gate met: package now at **54.1%** (resize-reflow work added
      more tests after the items above).

### Atomic state writes — closed the one bypass

- [x] Exported `state.Create` (locked, atomic temp+rename) and routed
      `workspace.writeStateYAML` through it with the canonical `state.State` struct.
- [x] (found during the fix) The hand-rolled marshal in `work.go` had drifted: the
      owner→worker rename (8ddf6c5) missed it, so scaffolds wrote `stage.owner` /
      `history[].owner` that `state.Load` silently dropped. Round-trip now covered
      by tests in both `state` and `workspace`.
- [x] (same root cause) The embedded templates still instructed agents to run
      `orc mark <ticket> advance --owner <id>` and `orc mark <ticket> wait|block` —
      none of which exist. Fixed to `next --worker` / `pause` across stage files,
      ORC.md, AGENTS.md sample worker, and feature README wording.

### `orc adhoc` — already implemented as `orc jit`

No work needed: `orc jit <ticket> --worker <id> "<instruction>"` implements every
decision in the adhoc spec (archive-aware lookup, no stage/status change, per-run
output dir, history entry on completion) plus refinements the spec lacked
(`runtime.jit` visibility in status/TUI, `orc mark <ticket> jit` to close,
concurrent-task blocking). The spec memory predated jit's implementation.

### `handleKey` per-view split (done 2026-06-11)

- [x] The 354-line `handleKey` is now a 15-line dispatcher over five per-view
      handlers (`handleDashboardKey`, `handleDetailKey`,
      `handleWorkflowDetailKey`, `handleFileKey`, `handleCharacterSheetKey`).
      Repetition extracted along the way: `toggleSection` (was four copies),
      `cycleSectionFocus`/`focusSection` (tab cycling), `openViewer` (was four
      copies of the file-viewer setup), `openSectionItem`. Behavior-preserving
      — the full `update_test.go` suite passes unchanged; coverage 54.1% →
      55.5%. The left/right stage-navigation roadmap item now lands in small
      focused handlers instead of a monolith.

### Release automation (done 2026-06-11)

- [x] Tags + releases: `make build` stamps the version from `git describe
      --tags` (was hardcoded `dev`); `.goreleaser.yaml` builds darwin/linux ×
      amd64/arm64 archives; `.github/workflows/release.yml` tests and
      publishes a GitHub release on any `v*` tag push. First tag: `v0.1.0`.

### CI (done 2026-06-11)

- [x] `.github/workflows/ci.yml` runs lint + test on push to main and on PRs.
      golangci-lint pinned to v2.12.2 (matching local), Go version from
      `go.mod`. Same gate as the pre-commit hook, no longer opt-in.

### Pre-commit hook fmt check (done 2026-06-11)

- [x] The gofmt check used `git diff --quiet` over the whole tree, so any
      unstaged file (docs, scratch notes) blocked a commit even when
      formatting was clean. Now keys off `gofmt -l` output directly — fails
      only when Go files actually need reformatting, and names them.

### `orc doctor --fix` (done 2026-06-11)

- [x] `--fix` removes provably-stale state locks — dead PID, or old without a
      valid PID, the same staleness definition the write path's auto-recovery
      uses (`state.ClearStaleLock`, exported from the existing
      `removeStaleLock` logic). Live or ambiguous locks are reported, never
      touched. Covered by tests in `state`, `doctor`, and `cmd/orc`;
      README updated.
- [x] (follow-up) `orc doctor <ticket> --fix` was silently a no-op — the flag
      only applied in workspace mode. Ticket mode now clears that ticket's
      stale lock before validation and prints what it removed.

### Small items

- [x] README dependencies: reworded — chafa is the character-art fallback; pixels
      come natively on kitty/Ghostty via Unicode placeholders.
- [x] `characters1.png` / `characters2.png` moved to
      `internal/tui/assets/portraits/sheets/` (source sheets: hero classes already
      cropped into portraits; orc classes available for future portraits). Outside
      the go:embed pattern, so binary size is unaffected.
- [x] (found during cleanup) Removed stray tracked file
      `internal/tui/assets/portraits/png/.png` — a crop-marked ranger intermediate
      saved with an empty filename, never embedded (go:embed skips dotfiles).
- [x] (found during cleanup) `kittyPortraitSupported` now checks tmux
      `allow-passthrough` and falls back to chafa when it's off — previously the
      placeholder grid would render as garbage because the image transmission was
      silently dropped.
