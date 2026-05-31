# Project Cleanup and Future Work

## Overall Assessment

`orc` has a strong core idea: treat agent work as a durable filesystem state
machine, not as a chat session. The feature folder, `STATE.yaml`, stage docs,
worker files, and `orc next` loop are practical and easy to reason about.

The project is strongest when it stays a CLI that coordinates files, prompts,
agents, and handoffs. It should not try to become a full AI platform. The durable
state and file-based policy model are the valuable pieces.

The main risk is that the system currently depends heavily on agent obedience.
The docs tell agents what to do, but `orc` does not yet enforce enough of the
contract. Required outputs, legal state transitions, stale sessions, missing
stage artifacts, malformed configs, and incomplete handoffs can all slip through.

The next level of maturity is making `orc` verify the contract, not just describe
it.

## Current Gaps

1. **Docs and code drift**

   Status: fixed for the stale `CLAUDE.md` items listed below.

   `CLAUDE.md` was stale in a few places:

   - It still lists `orc tui` as planned, even though the command and `internal/tui`
     exist.
   - It references `docs/` and `assets/` directories that are not currently present.
   - It omits `orc work --workflow`.
   - It does not mention `settings.default_workflow`.

   This kind of drift matters because agents will follow the generated docs
   literally.

2. **State transitions are under-validated**

   `state.Advance`, `state.WaitForHuman`, and `state.Block` update YAML, but there
   is not enough validation around whether the transition is legal.

   Missing checks include:

   - Current stage exists in the selected workflow.
   - Target stage exists.
   - Manual gates are not skipped accidentally.
   - Required outputs exist before advancing.
   - Repair stages respect `max_retries`.
   - `STATE.yaml` is internally consistent.

3. **Recovery is not first-class**

   If an agent dies mid-stage, `status: in_progress` can remain forever. There is
   no dedicated command to summarize partial work and produce a clean restart
   prompt.

   This is probably the most important missing product loop for real use.

4. **Agent-created worktree support needs validation**

   `orc archive` removes worktrees recorded in `STATE.yaml`, while agents are
   expected to create the worktrees during stages that need repository changes.
   That split is reasonable, but the contract needs validation.

   `orc` should verify that agent-created worktrees exist, match configured repo
   names, and are recorded consistently in `STATE.yaml`.

5. **Config parsing is split**

   `internal/config` and `internal/workflow` both read `orc.yaml`. This is workable
   now, but it will age poorly as settings grow.

   A single parsed workspace config with typed subsections would reduce drift and
   make validation easier.

6. **Command logic is concentrated in `cmd/orc/main.go`**

   `cmd/orc/main.go` currently owns CLI parsing, output rendering, workflow
   resolution, next prompt construction, worker selection, tmux orchestration,
   archive behavior, and state transitions.

   This makes behavior harder to test without invoking command globals.

7. **Worker matching semantics are ambiguous**

   Worker frontmatter supports both `workflows:` and `stages:`, but matching is
   effectively done against stage names. That can be confusing as named workflows
   become more important.

   Prefer making `stages:` the primary matching field and reserve `workflows:` for
   actual pipeline names, if needed.

## Prioritized Future Work

1. **Add `orc validate <ticket>`**

   Validate the current feature state and report actionable problems.

   Checks should include:

   - `STATE.yaml` parses and has required fields.
   - `workflow` exists in `orc.yaml`.
   - `stage.name` exists in the selected workflow or repair stages.
   - Stage markdown exists at `stages/<stage>.md`.
   - Current worker exists.
   - Required input/output files exist for the stage.
   - Transition hints are valid.
   - Runtime tmux state is either active or clearly stale.

2. **Add `orc resume <ticket>` or `orc recover <ticket>`**

   Generate a restart prompt when a session ends mid-stage without advancing.

   It should read:

   - `STATE.yaml`
   - History entries
   - `DECISIONS.md`
   - Current stage output folder
   - Recent known artifacts such as `develop/HANDOFF.md` or `code-review/REVIEW.md`

   Output should give the next agent a concise recovery context and a clear command
   to run at completion.

3. **Extract next-action planning from `cmd/orc/main.go`**

   Create a testable internal package, for example `internal/runner` or
   `internal/orc`, that computes:

   - Resolved workflow
   - Current stage config
   - Next stage
   - Completion instruction
   - Selected worker
   - Prompt
   - Launch args

   Keep Cobra functions focused on flags, arguments, and rendering.

4. **Validate and track agent-created worktrees**

   Keep worktree creation in the agent/stage layer, but make the tracking contract
   explicit and verifiable.

   Useful checks:

   - `repos.<name>.main` points at an existing main repo.
   - `repos.<name>.worktree` exists under workspace `worktrees/`.
   - The worktree branch matches `repos.<name>.branch`.
   - The repo name exists in `orc.yaml`.
   - `next_action.cwd` points at the active worktree when the next agent should continue there.
   - `orc archive` can remove every recorded worktree cleanly.

5. **Unify config parsing**

   Parse `orc.yaml` once into a single typed config that contains:

   - `settings`
   - `repos`
   - `workflows`
   - `repair_stages`

   Then make `internal/config` the owner of the full file, or rename the package to
   reflect the broader responsibility.

6. **Add schema/versioning**

   Add `schema_version` to `STATE.yaml` and potentially `orc.yaml`.

   This gives future changes a migration path and makes validation clearer.

7. **Tighten docs as executable contract**

   Add tests or a lightweight audit command to catch drift between docs/templates
   and the CLI.

   Useful checks:

   - Every `orc <command>` mentioned in templates exists.
   - README command table matches Cobra commands.
   - `CLAUDE.md` roadmap does not list implemented features as planned.
   - Template paths referenced in docs exist.

8. **Clarify worker routing**

   Make routing fields explicit:

   - `stages:` means stage names such as `develop` or `code-review`.
   - `workflows:` means workflow names such as `default` or `hotfix`.
   - Stage config `worker:` remains the preferred default.

   Tests should cover fallback order and ambiguous cases.

## Product Direction

The highest-leverage direction is reliability, not feature count.

Before adding many new UI or integration features, strengthen:

1. State validation
2. Recovery from interrupted sessions
3. Config consistency
4. Testable next-action planning
5. Docs/template drift checks

Those improvements make the tool trustworthy. Once that foundation is solid,
features like notifications, richer TUI actions, custom themes, and additional
agent products will be much easier to add without making the system brittle.
