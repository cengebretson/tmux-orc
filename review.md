# Go Code Review Notes

These notes focus on Go organization, tests, and maintainability risks from the current workspace state.

## Findings

1. **Inconsistent default workflow resolution**

   Status: mostly fixed. `runNextAction`, `runAdvance`, `runStatus`, and `runStart`
   now use the shared workflow resolver. A broader extraction out of `cmd/orc/main.go`
   is still worth doing later.

   `settings.default_workflow` is applied when creating new work in `internal/workspace/work.go`, but some CLI paths still fall back to the literal `"default"` when `STATE.yaml` has an empty `workflow`.

   Affected areas:

   - `cmd/orc/main.go`: `runNextAction` uses:

     ```go
     pname := s.Workflow
     if pname == "" {
     	pname = "default"
     }
     ```

   - `cmd/orc/main.go`: `runAdvance` has the same fallback before calling `workflowCfg.NextStage`.

   Impact:

   - Older/manual feature states with no `workflow` field can select the wrong workflow.
   - `orc next` JSON and normal `orc next` can disagree because JSON uses `resolveWorkflow`, while `runNextAction` does not.
   - `orc advance` can fail to move through the configured default workflow if the workspace default is not named `default`.

   Suggested fix:

   - Use one shared helper for workflow resolution everywhere, preferably outside `cmd/orc/main.go`.
   - Replace literal fallback logic with `resolveWorkflow(root, s.Workflow)` or a service-level equivalent.
   - Add tests for legacy state files with empty `workflow` and `settings.default_workflow` set.

2. **`workspace.Work` can leave a partial feature directory on config errors**

   Status: fixed. Config and workflow validation now happen before copying
   `features/_template`, with a regression test for missing workflows.

   `internal/workspace/work.go` copies `features/_template` into the new feature directory before loading and validating `orc.yaml` and the selected workflow.

   Impact:

   - If `orc.yaml` is invalid, unreadable, or the workflow is missing, `Work` returns an error after creating the feature folder.
   - A retry then fails as a duplicate feature, even though `STATE.yaml` may not have been written correctly.

   Suggested fix:

   - Load config and validate the workflow before copying the template directory.
   - Alternatively, clean up the just-created feature directory on failure, but validating first is simpler and safer.

3. **`cmd/orc/main.go` has too much command behavior**

   `cmd/orc/main.go` currently owns Cobra wiring, user-facing output, workflow resolution, next prompt construction, worker selection, tmux orchestration, archive behavior, and state transitions.

   Impact:

   - Core behavior is difficult to unit test without invoking command globals.
   - Similar logic is duplicated across JSON and non-JSON paths.
   - Naming drift is easier to introduce because stage/workflow concepts are handled ad hoc in multiple command functions.

   Suggested organization:

   - Keep `cmd/orc/main.go` focused on Cobra flags, argument parsing, and output.
   - Move reusable behavior into an internal package such as `internal/runner` or `internal/orc`.
   - Good extraction targets:
     - Resolve workflow for a state.
     - Build next prompt.
     - Determine next stage and completion instructions.
     - Select worker from explicit override, stage owner, workflow default, or capability match.

4. **Workflow names are nondeterministic**

   Status: fixed. `workflow.Config.Names()` now sorts names before returning them.

   `internal/workflow.Config.Names()` returns map keys directly.

   Impact:

   - Error output like `available: ...` can change order between runs.
   - Health output can be nondeterministic.
   - Tests that assert ordering would be flaky.

   Suggested fix:

   - Sort the returned names in `Names()`.

5. **Tests do not cover the new config/default workflow behavior**

   Status: partially fixed. Workspace tests now cover configured default workflow,
   explicit workflow override, and invalid workflow cleanup. CLI-level legacy
   state tests are still missing.

   Current workspace tests verify that feature folders and `STATE.yaml` are created, but they do not inspect the written workflow or first stage.

   Suggested tests:

   - `workspace.Work` uses `settings.default_workflow` when no `WorkOptions.Workflow` is provided.
   - Explicit `WorkOptions.Workflow` overrides `settings.default_workflow`.
   - Missing workflow returns a useful error and does not leave a feature directory behind.
   - Legacy state with empty `workflow` advances and launches using the configured default workflow.

6. **Stage/workflow naming is muddy**

   Several places label a stage as a workflow. For example, `runShow` prints `Workflow: s.Stage.Name`, and `state.Advance(featureDir, workflow, ...)` takes a parameter named `workflow` even though it represents the next stage name.

   Impact:

   - Makes future workflow work harder to reason about.
   - Increases the chance of regressions when adding multiple workflows or default workflow settings.

   Suggested fix:

   - Use `workflow` only for pipeline names.
   - Use `stage` or `stageName` for individual steps.
   - Rename parameters and output labels opportunistically when touching related code.

## Suggested Follow-Up Order

1. Extract next-action planning and worker selection out of `cmd/orc/main.go`.
2. Add CLI/service tests for legacy states with empty `workflow`.
3. Clean up remaining stage/workflow naming as related code is touched.
4. Add broader validation around `STATE.yaml`, workflow transitions, and required outputs.

## Verification

Tests pass with the Go build cache redirected into the workspace:

```sh
GOCACHE=/private/tmp/orc-gocache go test ./...
```

The normal `go test ./...` command failed in this sandbox because the default macOS Go build cache path was not writable.
