# Re-Review Notes

These notes cover the current workspace after the recent cleanup work. Focus is on
remaining bugs, regressions, and test gaps rather than restating completed fixes.

## Findings

1. **Config load errors can still panic validation and advance paths**

   `internal/validate/validate.go` ignores `config.Load(root)` errors, then
   dereferences the returned config when checking workflow/stage data. If
   `orc.yaml` is malformed or unreadable, `orc validate <ticket>` can panic
   instead of returning a failed validation check.

   Affected areas:

   - `internal/validate/validate.go`: `cfg, _ := config.Load(root)`
   - `internal/validate/validate.go`: `wfCfg, _ := config.Load(root)`
   - `cmd/orc/main.go`: `workflowCfg, _ := config.Load(root)` in `runAdvance`

   Impact:

   - `orc validate` is supposed to be the safe diagnostic path, but a malformed
     config can crash it.
   - `orc advance` can also panic before returning an actionable error.

   Suggested fix:

   - Load config once and handle the error explicitly.
   - In `validate.Run`, append a failed `orc.yaml` or `workflow` check and return
     the report.
   - In `runAdvance`, return `fmt.Errorf("loading config: %w", err)`.
   - Add tests for malformed `orc.yaml` in both validate and advance-adjacent
     logic where practical.

2. **Transition guard does not require recorded worktrees to exist**

   `state.ValidateRepos` checks that recorded worktree paths are under the
   workspace `worktrees/` directory and that a branch is recorded, but it does not
   check that the worktree path actually exists.

   Affected areas:

   - `internal/state/state.go`: `ValidateRepos`
   - `cmd/orc/main.go`: `runAdvance` and `runWait` rely on `ValidateRepos`

   Impact:

   - A ticket can advance or wait with a stale/missing recorded worktree.
   - `orc validate` reports missing worktrees separately, so the behavior is
     inconsistent between validation and lifecycle commands.

   Suggested fix:

   - Add an existence check for each non-empty `repos.<name>.worktree`.
   - Keep the current "no worktree recorded is allowed" behavior.
   - Add a regression test for a missing recorded worktree.

3. **`orc show` still labels the current stage as workflow**

   `runShow` prints `Workflow: s.Stage.Name` in the `Stage` section.

   Affected area:

   - `cmd/orc/main.go`: `fmt.Printf("  Workflow:  %s\n", s.Stage.Name)`

   Impact:

   - This keeps the stage/workflow naming ambiguity alive in user-facing output.
   - It is minor, but it works against the multi-workflow cleanup.

   Suggested fix:

   - Print the resolved workflow separately if useful.
   - Label the current stage as `Stage:`.

## Status

The recent changes are a substantial improvement:

- Config and workflow parsing are unified under `internal/config`.
- Next-action planning has moved into `internal/runner`.
- Runner behavior has focused tests.
- Default workflow behavior is cleaner.
- `orc validate` and `orc resume` establish useful reliability loops.
- Worktree contract validation has started.

## Verification

The full Go suite passes with the build cache redirected into a writable path:

```sh
GOCACHE=/private/tmp/orc-gocache go test ./...
```

The normal `go test ./...` command may still fail in this sandbox if the default
macOS Go build cache path is not writable.
