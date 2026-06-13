# orc — Plan

Single roadmap file. **Up next** holds specced, ready-to-build work;
**Future ideas** are unspecced and lower priority; **Cleanup** is open
hardening items; the **Done** record and **Considered and rejected** keep
decisions from being relitigated.

---

## Up next

### Agent completion notification

Fire a user-defined shell command when a ticket transitions state. Most useful
in `--tmux` mode where sessions run unattended and the user needs a signal that
work is ready for review or is blocked.

look at slackcat for an example: `slackcat --channel alerts --username Grommash -m "The build is complete, warchief."`

#### Config shape (`orc.yaml`)

```yaml
settings:
  notify:
    on: [blocked, complete]          # events: blocked | complete | error | all
    command: "notify-send 'orc' '{{ticket}} {{event}}'"
```

Event data is exported as environment variables (`ORC_TICKET`, `ORC_SLUG`,
`ORC_EVENT`, `ORC_STAGE`, `ORC_WORKFLOW`) and the command runs as-is — no
string splicing into shell, so quoting can never break. `{{var}}` template
expansion is kept as sugar for the same five values:
- `{{ticket}}` — ticket ID (e.g. `STORY-123`)
- `{{slug}}` — full feature slug (e.g. `STORY-123-add-login`)
- `{{event}}` — event name (`blocked`, `complete`, `error`)
- `{{stage}}` — stage name at the time of the event
- `{{workflow}}` — workflow name

#### Events

| Event | When it fires |
|-------|--------------|
| `complete` | ticket advances to next stage (or done if final) |
| `blocked` | `orc mark <ticket> pause` is called |
| `error` | future — agent explicitly signals failure |
| `all` | shorthand for all of the above |

#### Implementation notes

- Add `NotifySettings` struct to `internal/config` with `On []string` and `Command string`
- Add `Notify NotifySettings yaml:"notify"` to `Settings`
- Add `internal/notify` package: `Fire(cfg *config.NotifySettings, event, ticket, slug, stage, workflow string)` — sets `ORC_*` env vars, expands template vars, checks `On` list, runs command via `os/exec` with a short timeout
- Call `notify.Fire` in `runMarkNext` and in `runMark` for both the **pause** and **done** cases after state is written — `orc mark <ticket> done` is its own switch arm and must fire `complete` too, not just the advance path
- No-op when `command` is empty or event not in `on` list

**Effort:** Medium.

---

### Per-run log capture — `orc logs` / `orc tail`

The design principles name logging as the prerequisite for background
execution ("Background execution comes after logging and recovery are
solid"), and no command covers it today. Capture every launched agent's
transcript into the stage's output folder so runs are reviewable after the
fact.

- **tmux mode**: after session create in `internal/orchestrator/launch.go`,
  run `tmux pipe-pane -o 'cat >> <featureDir>/<stage>/run-<timestamp>.log'`
  so the full pane transcript streams to disk.
- **direct mode**: tee the launched process's stdout/stderr to the same path.
- Record the active log path under `runtime` in `STATE.yaml` (same pattern as
  `runtime.tmux.session`) so `status` and the TUI can surface it.
- `orc logs <ticket>` prints the most recent log (`--stage` to pick a stage);
  `orc tail <ticket>` follows the active run's log.
- Logs live inside the feature folder, so `orc archive` preserves them for
  free — evidence collection without extra plumbing.

**Effort:** Medium.

---

## Future ideas

Lower priority — worth revisiting once the core is solid.

- **Workspace packs** — share workers/workflows across a team. `orc pack
  push/pull/diff` — copy workers, stages, RULES.md to/from a git repo;
  two-layer model with `overrides/` for local customization.
- **Per-worker cost attribution** — build on `orc report`: history entries
  carry `worker`, worker definitions carry model/cost tier; roll stage
  durations up into per-worker time and estimated cost.
- **TUI event feed** — once `notify.Fire` lands (Up next above), also append
  events to a workspace `events.log`; the TUI tails it into a recent-activity
  pane. Cheap after notifications ship; pointless before.

---

## Cleanup

- [ ] (optional) `viewDashboard` composition remains untested (`viewDetail`
      now has a smoke test via the Timing section); golden-file snapshot tests
      are the cheapest way in if it ever matters.

---

## Considered and rejected

- `internal/stage` at 0% coverage — 36 trivial lines, not worth tests.
- `internal/tmux` at 23% — wraps the tmux binary; mocking it buys little.
- `cmd/orc/main.go` at 1,388 lines — it's flag definitions and printing, which is
  exactly what the "CLI as the boundary" principle prescribes. No action.
- Cobra package-level flag vars (`markStage` etc.) — standard cobra idiom.

---

## Done — roadmap items

### Stage timing — `orc report` (done 2026-06-12)

- [x] New `internal/report` package: `Compute(*state.State, now) Report` derives
      per-stage active/wall/visits from history (each entry stamps the stage
      active up to its time, so an interval is attributed to the closing entry's
      stage; the open current stage is measured to `now`). `Aggregate([]Report)`
      rolls up avg/median active + visit counts per stage. Pause→resume gaps are
      subtracted from active time; bad/out-of-order timestamps are skipped, never
      fatal. Terminal statuses (`done`, `archived`) get no open interval; a
      `paused` ticket's current-stage interval is frozen via the status field
      (authoritative over the free-form result string). Package coverage ~97%.
- [x] `orc report [ticket]`: no-arg aggregate table across tickets (`--archived`
      to include `_archive/`), `orc report <ticket>` single-ticket breakdown with
      total cycle time, both with `--json`. Thin handler in `cmd/orc`; tickets
      collected via `featurelist.Collect`. Command-level tests cover table,
      JSON shape, and aggregate.
- [x] TUI: the feature detail page gained a **Timing** section above History,
      rendering `report.Compute` (per-stage active/wall/visits + total, with a
      "← current" marker on the open stage). First test to exercise `viewDetail`
      directly.
- Defaults shipped as specced: active time is the headline (wall shown
  alongside), repair loops summed per stage with a visit count. The command
  was the first surface; the TUI Timing section followed. Per-worker cost
  attribution remains in Future ideas, building on this.

### TUI left/right stage navigation (done 2026-06-12)

- [x] Stage file viewer: `left`/`right` (`h`/`l`) jump to the previous/next
      stage's `.md` in pipeline order, driving `wfDetailCursor` and rebuilding
      the `step N of M` title (`loadViewerStage`, mirror of `loadViewerFile`).
      Esc back now scrolls the workflow detail so the cursor row stays visible
      (was restoring the file viewer's offset). Workflow detail page:
      `left`/`right` alias the stage cursor. Help text updated on both views.
      Coverage 55.5% → 58.1%.

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
would catch. The gate was "do this before the rest of the TUI roadmap" —
coverage is now 54.1% with the file split done, so the roadmap is unblocked.
One soft follow-up remains under Cleanup above.

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
