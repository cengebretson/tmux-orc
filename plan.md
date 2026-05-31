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

Fire a user-defined shell command when a ticket transitions state. Most useful
in `--tmux` mode where sessions run unattended and the user needs a signal that
work is ready for review or is blocked.

#### Config shape (`orc.yaml`)

```yaml
settings:
  notify:
    on: [blocked, complete]          # events: blocked | complete | error | all
    command: "notify-send 'orc' '{{ticket}} {{event}}'"
```

Template variables available in `command`:
- `{{ticket}}` — ticket ID (e.g. `STORY-123`)
- `{{slug}}` — full feature slug (e.g. `STORY-123-add-login`)
- `{{event}}` — event name (`blocked`, `complete`, `error`)
- `{{stage}}` — stage name at the time of the event
- `{{workflow}}` — workflow name

#### Events

| Event | When it fires |
|-------|--------------|
| `complete` | `orc advance` moves to the next stage (or archives if last) |
| `blocked` | `orc wait` writes a `waiting_for_human` or `blocked` status |
| `error` | future — agent explicitly signals failure |
| `all` | shorthand for all of the above |

#### Implementation notes

- Add `NotifySettings` struct to `internal/config` with `On []string` and `Command string`
- Add `Notify NotifySettings yaml:"notify"` to `Settings`
- Add `internal/notify` package: `Fire(cfg *config.NotifySettings, event, ticket, slug, stage, workflow string)` — expands template vars, checks `On` list, runs command via `os/exec` with a short timeout
- Call `notify.Fire` in `runAdvance` and `runWait` in `cmd/orc/main.go` after state is written
- No-op when `command` is empty or event not in `on` list

**Effort:** Medium.

---

## Future ideas

Lower priority — worth revisiting once the core is solid.

| Idea | Notes |
|------|-------|
| ~~Quotes in `orc.yaml`~~ ✓ Done | `settings.quotes: [...]` — troll quotes ship as default in the workspace template. |
| ~~Rainbow logo easter egg~~ ✓ Done | Type `orc` on the dashboard — logo and header title cycle through all 12 Catppuccin palette colors for ~4 seconds. |
| Bard's Tale character sheet easter egg | Press `!` on the worker detail page to reveal a retro RPG character sheet. See spec below. |
| ~~Theme configuration~~ ✓ Done | Colors extracted to `internal/tui/themes/catppuccin-mocha.json`, `LoadTheme()` reads it at startup, `settings.theme` in orc.yaml controls which file loads. |
| ~~Ticket system config~~ ✓ Done | Moved to `ROUTER.md` — a dedicated **Ticket System** section tells agents where to fetch tickets. Keeps it with repo routing info where it belongs. |

---

### Bard's Tale character sheet easter egg (spec)

Press `!` on any worker detail page to toggle a retro RPG character sheet overlay.
Press `!` again (or `esc`) to return to the normal worker view.

#### Layout

```
┌─ CHARACTER SHEET ──────────────────┐
│  ┌──────┐  Name:  Bob (Developer)  │
│  │ ASCII│  Class: WARRIOR          │
│  │  art │  Race:  Claude           │
│  │      │  Level: opus-4           │
│  └──────┘  Guild: bob-the-developer│
├────────────┬───────────────────────┤
│ ST ████░░  │ WEAPON  claude-opus-4 │
│ IQ ███████ │ SHIELD  auto-advance  │
│ DE ████░░  │ HELM    develop       │
│ CN █████░  │ RING    —             │
│ LK ██░░░░  ├───────────────────────┤
│            │ HP ████  3 quests     │
│            │ XP ██░░  12 complete  │
├────────────┴───────────────────────┤
│ Active quests:                     │
│  ► STORY-123  develop  in_progress │
│    FLYWL-099  review   waiting     │
└────────────────────────────────────┘
```

#### Data mappings

| Sheet field | Source |
|-------------|--------|
| Name | worker `name:` |
| Class | derived from name keywords: Developer→WARRIOR, QA→RANGER, Document→BARD, Ninja→ROGUE, default→ADVENTURER |
| Race | worker `engine:` field (claude→CLAUDE, codex→CODEX, cursor→CURSOR) |
| Level | worker `model:` field |
| Guild | worker ID (filename stem) |
| ST / IQ / DE / CN / LK | deterministic from worker ID hash — same worker always has same stats |
| WEAPON | model name |
| SHIELD | advance mode most common in their assigned stages (auto/manual) |
| HELM | first stage name found in orc.yaml assigned to this worker |
| HP | count of active tickets assigned to this worker |
| XP | count of history entries in STATE.yaml files owned by this worker |
| Active quests | live ticket list (ticket, stage, status) for this worker |

#### Portrait system

Small ASCII art portraits stored as a slice of `[]string` in a new `internal/tui/portraits.go` file.
Portraits are grouped by class (warrior, ranger, bard, rogue, generic pool).
Portrait is selected by: `portraits[classPool][hash(workerID) % len(classPool)]` — deterministic,
so the same worker always shows the same face.

Each portrait fits in ~8 lines × 12 chars to fill the top-left box.

Example warrior portrait:
```
   O
  /|\
  / \
 sword
```

Ship at least 3 portraits per class (warrior, ranger, bard, rogue) + 5 generic fallbacks.

#### Visual style

- Box-drawing borders (`┌─┬─┐│└─┴─┘`)
- Surface0 background, Yellow for stat bars (`█` filled, `░` empty)
- Mauve for section headers, Text for values
- Stat bars: 8 chars wide, value 1–20

#### Implementation notes

- Add `viewCharacterSheet` to the `viewState` enum
- Add `charSheetWorker *workers.Worker` to Model
- `!` in `viewDetail` (worker detail) sets `m.view = viewCharacterSheet` and `m.charSheetWorker`
- `!` or `esc` in `viewCharacterSheet` returns to `viewDetail`
- `renderCharacterSheet(w *workers.Worker, features []*featureRow, width int) string` builds the full sheet
- Stats derived via `workerStats(id string) [5]int` using FNV hash of the ID, values 5–18 range
- New file `internal/tui/portraits.go` — portrait data only, no logic

**Effort:** Medium-high. Mostly rendering code; no state changes required.

---

~~### Theme configuration~~ ✓ Done

Implemented: palette and glamour style extracted to `internal/tui/themes/catppuccin-mocha.json`.
`LoadTheme(name)` reads the JSON at startup, `initStyles()` reinitializes all style vars.
`settings.theme` in orc.yaml selects the theme file; defaults to `catppuccin-mocha`.
