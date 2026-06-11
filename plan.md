# orc — Plan

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

### TUI: left/right stage navigation on workflow + stage views

The feature detail page already cycles its markdown files with `left`/`right`
(`h`/`l`), including inside the file viewer. Bring the same navigation to the
workflow side:

- **Stage file viewer** (opened with `enter` from a workflow detail page):
  `left`/`right` should jump to the previous/next stage's `.md` in pipeline
  order, updating the `step N of M` title — mirror of the `viewerReturn ==
  viewDetail` branch in `viewFile` (`internal/tui/update.go`), driving
  `wfDetailCursor` instead of `fileIdx`.
- **Workflow detail page**: `left`/`right` as an alias for the existing
  `up`/`down` stage cursor movement (steps render as a horizontal chain, so
  horizontal keys are the natural ask).

**Effort:** Small.

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

Lower-priority future ideas live in `todo.md` alongside cleanup and hardening
items; this file holds only specced, up-next work.
