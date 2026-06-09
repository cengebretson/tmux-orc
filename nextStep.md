# orc Next Steps

This file turns the architecture review into actionable work. The goal is to harden
the file-backed workflow model without turning `orc` into a hidden workflow engine.

## 1. Define the `STATE.yaml` Contract

**Why:** `STATE.yaml` is the durable source of truth for every feature. As the
tool grows, the state file needs an explicit ownership and compatibility model.

**Actions:**

- Document every top-level `STATE.yaml` field.
- Mark each field as one of:
  - `orc-owned`
  - `agent-writable`
  - `human-editable`
  - `derived/runtime`
- Add a `schema_version` field to new feature state files.
- Decide whether legacy state files without `schema_version` are treated as v1.
- Add validation for unknown or malformed required fields.

**Acceptance criteria:**

- `internal/workspace/templates/features/_template/STATE.yaml` includes `schema_version`.
- `internal/workspace/templates/ORC.md` explains state ownership rules.
- `orc health <ticket>` reports actionable errors for invalid state.
- Existing test fixtures still load successfully.

## 2. Formalize Workflow Semantics in `orc.yaml`

**Why:** Workflow policy should live in workspace files, but `orc` still needs to
enforce generic transition rules consistently.

**Actions:**

- Document the supported workflow fields:
  - `settings.default_workflow`
  - `settings.auto_archive`
  - `workflows.<name>.stages`
  - stage `name`
  - stage `worker`
  - stage `advance`
  - stage `loop`
- Define valid values for `advance`, `loop.on_max`, and status transitions.
- Add validation for missing worker IDs referenced by workflow stages.
- Add validation for loop stages that point to missing owner stages.
- Add validation for workflows with duplicate stage names.

**Acceptance criteria:**

- `README.md` or a dedicated `docs/workflows.md` contains a complete `orc.yaml` reference.
- Invalid workflow config fails clearly in `orc doctor` or `orc health`.
- Tests cover missing workers, duplicate stages, and invalid loop targets.

## 3. Keep `runner.Compute` as the Single Next-Action Resolver

**Why:** The CLI, TUI, dry-run output, tmux launch path, and future automation all
need the same answer to “what happens next?”

**Actions:**

- Audit command handlers for duplicated next-stage, worker, or prompt resolution.
- Move duplicated resolution logic into `internal/runner`.
- Add table-driven tests for:
  - workflow default resolution
  - stage worker resolution
  - explicit worker override
  - loop-stage re-entry
  - manual vs automatic advance instructions
- Consider exposing a lightweight JSON shape for resolved plans.

**Acceptance criteria:**

- `orc next <ticket> --dry` and any TUI/preview path use the same resolved plan.
- New next-action behavior can be tested through `runner.Compute`.
- No command handler independently reconstructs stage/worker/prompt behavior.

## 4. Strengthen State Mutation Safety

**Why:** File-backed coordination only works if state updates are atomic,
recoverable, and clear when something is stale or locked.

**Actions:**

- Keep `state.Update` as the only write path for `STATE.yaml`.
- Document lock behavior for `STATE.yaml.lock`.
- Add tests for concurrent update attempts where practical.
- Add clearer doctor output for stale locks, live locks, and lock cleanup.
- Decide whether lock files should include PID, command, hostname, and timestamp.

**Acceptance criteria:**

- All state-writing code paths use `state.Update` or a small wrapper around it.
- `orc doctor` explains what to do when a lock is stale or active.
- Tests prove failed mutations do not rewrite state.

## 5. Preserve the CLI/TUI Boundary

**Why:** The TUI should present workflow state, not become a parallel workflow
implementation.

**Actions:**

- Keep presentation aggregation in `internal/featurelist` or another shared view package.
- Avoid adding transition logic directly to `internal/tui`.
- Add tests around feature collection for workflow labels, workers, archived state, and errors.
- If TUI actions mutate state later, route them through the same orchestrator/state services as CLI commands.

**Acceptance criteria:**

- TUI state display is backed by shared feature collection logic.
- TUI mutation actions, if added, call existing domain services.
- CLI and TUI show consistent workflow/stage/worker information.

## 6. Reframe the Architecture Language

**Why:** “`orc` does not encode workflow logic” is directionally useful, but not
quite precise. The tool necessarily enforces generic transition behavior.

**Actions:**

- Update project docs to say:
  - workflow policy lives in workspace files
  - `orc` enforces generic state transitions and safety rules
  - agents write outputs and mark progress through the documented contract
- Make the boundary explicit in `README.md`, `CLAUDE.md`, and workspace templates.

**Acceptance criteria:**

- Docs no longer imply `orc` is logic-free.
- The policy-vs-mechanics boundary is explained consistently.
- New contributors can tell where to add workflow policy vs code-level safety checks.

## Suggested Order

1. `STATE.yaml` contract and ownership rules.
2. `orc.yaml` workflow reference and validation.
3. `runner.Compute` consolidation and tests.
4. State mutation safety improvements.
5. TUI boundary cleanup as new TUI behavior is added.
6. Documentation wording pass.

## First Concrete Ticket

Start with the `STATE.yaml` contract because it is the foundation for every other
piece of the architecture.

**Ticket shape:**

- Add `schema_version` to new feature templates.
- Document state ownership in `ORC.md`.
- Treat missing `schema_version` as legacy v1.
- Add validation coverage for required fields and malformed stage state.
- Keep existing fixtures passing.
