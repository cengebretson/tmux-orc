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

Template variables available in `command`:
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
- Add `internal/notify` package: `Fire(cfg *config.NotifySettings, event, ticket, slug, stage, workflow string)` — expands template vars, checks `On` list, runs command via `os/exec` with a short timeout
- Call `notify.Fire` in `runMarkNext` and `runMark` (pause case) after state is written
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

## Future ideas

Lower priority — worth revisiting once the core is solid.

| Idea | Notes |
|------|-------|
| Workspace packs — share workers/workflows across a team | `orc pack push/pull/diff` — copy workers, stages, RULES.md to/from a git repo; two-layer model with `overrides/` for local customization |
