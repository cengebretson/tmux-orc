# orc — Plan

---

## Human TODO

- [ ] Rename GitHub repo from `tmux-orc` → `orc` (GitHub Settings → Rename)
- [ ] After rename: update `go.mod` module path from `github.com/cengebretson/orc` to match (should already be correct if repo is renamed to `orc`)
- [ ] Update git remote locally: `git remote set-url origin git@github.com:cengebretson/orc.git`
- [ ] Verify `go install github.com/cengebretson/orc/cmd/orc@latest` works after rename

---

## Agent TODO

- [x] `orc delete <ticket>` — permanently remove a feature folder; only allowed when status is `done` or `archived`

---

## Up next

### `orc jit <ticket> --worker <id> "<instruction>"` (spec)

A JIT stage — same mechanics as a normal stage (worker, prompt, output dir, history entry) but
conjured on demand rather than declared in `orc.yaml`. Does not advance the pipeline stage.
The agent signals completion via `orc mark <ticket> jit-done`, which writes a history entry
so `orc status` shows the task happened.
Useful for spot checks, secondary reviews, or exploratory tasks that don't belong in the pipeline.

#### Command

```
orc jit <ticket> --worker <id> "<instruction>"
```

Flags:
- `--worker <id>` — required; the worker to run the task
- `--dry` — print the resolved worker and prompt without launching
- `--tmux` — run in a tmux window (uses the ticket's existing session if one is active)

#### Behavior

1. Resolve the feature dir (searches both `features/` and `features/_archive/`)
2. Resolve the named worker from `workers/` — error if not found
3. Build the prompt from the instruction + ticket context (see below)
4. Create output dir `features/<slug>/jit/<timestamp>/` — the agent writes here
5. Write `runtime.jit` to STATE.yaml: `{worker: <id>, task: <instruction>, started_at: <timestamp>}`
6. Launch the worker via the same `launchPlan` path used by `orc next`

Works against any ticket status (`pending`, `active`, `paused`, `done`, `archived`) — no guard.

#### CWD and orientation

CWD is set to the feature dir (`features/<slug>/`). The agent starts there, reads `STATE.yaml`
to understand the ticket, then navigates the workspace from there. No worktree resolution —
jit tasks are not tied to a specific repo operation.

#### Prompt shape

```
Before starting: read AGENTS.md and ORC.md.

## JIT task: <ticket>

<instruction>

## Context

Start in `features/<slug>/` and orient yourself by reading:
- `STATE.yaml` — current state and history
- `TICKET.md` — original ticket
- `SPEC.md` — scope and requirements (if present)
- `DECISIONS.md` — decisions made so far (if present)

Current pipeline stage: <stage> (do not advance — this is a one-off task outside the pipeline)

Write any output or notes to `features/<slug>/jit/<timestamp>/`.

When you are done, run:
  orc mark <ticket> jit-done "<summary of what you did>"
```

#### `orc mark <ticket> jit-done "<summary>"`

New subcommand of `orc mark` (hidden, agent-facing). Does three things only:
1. Appends a history entry: stage `"jit"`, worker ID, result = summary
2. Clears `runtime.jit` from STATE.yaml
3. Prints `Done: jit task recorded for <ticket>`

Does NOT change ticket status or pipeline stage.

#### STATE.yaml shape

```yaml
runtime:
  jit:
    worker: bob-the-developer
    task: "make sure fred did a good job"
    started_at: "2026-05-31T14:23:00Z"
```

Present while the jit job is running, absent otherwise. `orc status` shows it as a second
active item alongside the pipeline stage. The TUI surfaces it in two places:
- **Feature list row** — e.g. `develop + jit` or a small indicator so it's visible at a glance
  without obscuring the pipeline stage
- **Detail view** — full `runtime.jit` block shown as a distinct section: worker, task text,
  and started_at, so you can see exactly what's running and when it started

#### State impact

- Pipeline stage and status are unchanged throughout
- `runtime.jit` is set before launch, cleared by `jit-done`
- One history entry lands in STATE.yaml when the agent calls `jit-done` — visible in `orc status`

#### Implementation notes

- Add `jitCmd` (human, visible) and `runJIT` to `cmd/orc/main.go`
- Add `jit-done` branch to `runMark` in `cmd/orc/main.go` (alongside `next`, `pause`, `done`)
- Worker resolution: `workers.FindByID` directly — `--worker` is required, no fallback chain
- Prompt: `buildJITPrompt(s *state.State, workerID, instruction, outputDir string) string` — inline in `runJIT` or in `runner`
- Output dir: `filepath.Join(featureDir, "jit", time.Now().Format("20060102-150405"))`, created with `os.MkdirAll` before launch
- Feature dir: reuse `findFeatureDirWithArchive` (currently in `cmd/orc/main.go` — consider moving to `internal/state` so both commands share it)
- STATE.yaml: add `JITRuntime` struct and `runtime.jit` field to the `Runtime` struct in `internal/state`; add `state.SetJIT` and `state.ClearJIT` functions
- History: add `state.AppendHistory(featureDir, stage, workerID, result string) error` to `internal/state` — thin wrapper that loads, appends, and saves without touching any other fields

**Effort:** Small-medium. No new packages. Reuses `launchPlan`, `workers.FindByID`, and existing launch infrastructure. New surface area: `buildJITPrompt`, `JITRuntime` struct, `state.SetJIT`/`ClearJIT`, `state.AppendHistory`, and the `jit-done` mark branch.

### Helpful plugins

Add a list of mcp/plugins that help this workflow work. For example context-mode will save on tokens. I am sure there are others.

### autocomplete help

does autocomplete filter active tickets on tab complete?

### Agent completion notification

Fire a user-defined shell command when a ticket transitions state. Most useful
in `--tmux` mode where sessions run unattended and the user needs a signal that
work is ready for review or is blocked.

look at slackcate for an eample slackcat --channel alerts --username Grommash -m "The build is complete, warchief."

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
| Workspace packs — share workers/workflows across a team | `orc pack push/pull` or `orc pack apply <repo>` — see spec below. |
| Workspace packs — share workers/workflows across a team | `orc pack push/pull` — see spec below. |
| Bard's Tale character sheet easter egg | Press `!` on the worker detail page to reveal a retro RPG character sheet. See spec below. |

### Workspace packs — share workers, workflows, and policy across a team (spec)

A **pack** is a git repo (or subdirectory of one) that contains the shareable parts
of a workspace. Teams version it centrally; individuals pull it into their local workspace.

#### What goes in a pack

| File / directory | Shareable? |
|-----------------|-----------|
| `workers/*.md` | Yes — worker definitions are pure policy |
| `workflows/` (from `orc.yaml`) | Yes — pipeline shape and stage assignments |
| `stages/*.md` | Yes — stage instructions |
| `RULES.md` | Yes — approval policy is team-wide |
| `ROUTER.md` | Partial — repo paths are local; ticket system section is shareable |
| `orc.yaml` settings block | Partial — `default_workflow`, quotes; not local paths |
| `features/` | No — ticket work is always local |

#### Commands

```
orc pack pull <source>   # apply a pack into the current workspace
orc pack push <dest>     # copy shareable files out to a pack repo
orc pack diff <source>   # show what would change before pulling
```

`<source>` / `<dest>` is a local path or a git URL (plain clone, no branch pinning needed initially).

For a git URL, `orc pack pull` does a shallow clone to a temp dir, then applies. No permanent
remote tracking — it's a one-shot copy, not a sync relationship. The workspace stays
self-contained.

#### Two-layer model: pack + user overrides

Pack files live in their normal locations. User overrides live in a parallel
`overrides/` directory that mirrors the same structure:

```
workers/           ← pack-managed (replaced on pull)
stages/            ← pack-managed (replaced on pull)
RULES.md           ← pack-managed (replaced on pull)
overrides/
  workers/         ← user-owned, never touched by pack operations
  stages/
  RULES.md
```

`orc` resolves files by checking `overrides/` first, then falling back to the pack
file. This means a pull is always a clean replace of the pack layer — no merge
logic, no prompts, no risk of clobbering local changes. Users put customizations
in `overrides/` and they survive every pull automatically.

`orc init` and `SETUP.md` explain the convention. `orc health` can warn if an
override file shadows a pack file that has diverged significantly (future).

#### Apply behavior (`pull`)

- Replace `workers/*.md`, `stages/*.md`, `RULES.md` from the pack — no prompting.
- Merge `orc.yaml` workflows block: add new entries, leave existing ones alone.
- Merge `orc.yaml` settings named keys (`default_workflow`, `quotes`, `theme`), skip `repos`.
- Never touch `overrides/` or `ROUTER.md` — those are always user-owned.

#### Push behavior

- Copies `workers/`, `stages/`, `RULES.md`, and the workflows block from `orc.yaml` to `<dest>`.
- Strips any local-path fields before writing.
- If `<dest>` is a git repo, `orc pack push` stages the files but does NOT commit —
  leaves committing to the user.

#### `orc.yaml` pack source (optional)

```yaml
settings:
  pack: https://github.com/myteam/orc-pack.git   # or a local path
```

When set, `orc pack pull` with no args uses this source. Makes it easy to re-sync
after the team updates the pack.

#### Non-goals (keep it simple)

- No versioning / lockfile — it's a copy, not a dependency manager.
- No conflict resolution — the two-layer model eliminates the problem entirely.
- No auto-pull on `orc init` — explicit opt-in only.
- No private field encryption — sensitive credentials stay out of packs entirely.

**Effort:** Medium. Mostly file I/O and a simple YAML merge; the git-URL path adds a `git clone --depth 1` subprocess.

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

