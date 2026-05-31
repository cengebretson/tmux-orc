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
| Theme configuration | See spec below. |
| Ticket system config | `settings.ticket_system` for machine-readable source — lets the intake stage know where to fetch ticket data without hardcoding it in the stage doc. |

---

### Theme configuration (spec)

Swap the TUI color palette via `settings.theme` in `orc.yaml`. The hardcoded
Catppuccin Mocha constants in `tui.go` would be extracted into a theme struct
and resolved at startup.

#### Config shape

```yaml
settings:
  theme: catppuccin-mocha   # default; options: catppuccin-mocha | catppuccin-latte | dracula | gruvbox
```

#### Built-in themes to ship

| Name | Background | Feel |
|------|-----------|------|
| `catppuccin-mocha` | dark | current default |
| `catppuccin-latte` | light | same palette, light mode |
| `dracula` | dark | purple/pink accent |
| `gruvbox` | dark | warm retro |

#### Implementation notes

- Extract the color constants at the top of `tui.go` into a `Theme` struct
  with fields for each semantic role (`base`, `surface`, `text`, `subtext`,
  `mauve`, `green`, `yellow`, `red`, etc.)
- Add `var themes = map[string]Theme{...}` with the built-in palettes
- Add `Theme string yaml:"theme"` to `Settings` in `internal/config`
- Resolve the active theme in `Run()` and pass it into `New(root, theme)`
- All `lipgloss.NewStyle().Foreground(lipgloss.Color(mauve))` calls reference
  the resolved theme instead of the package-level constants

**Effort:** Medium — mechanical but thorough; touches every style definition in `tui.go`.
