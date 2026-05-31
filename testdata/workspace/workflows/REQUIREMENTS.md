# Workflow Requirements

Read this file before executing any workflow stage. It defines the orc state
contract that all workflows must follow.

---

## Status Values

Every feature has a `status` field in `STATE.yaml`. Keep it current at all times.

| Status | Meaning |
|--------|---------|
| `pending` | Feature created, intake has not run yet |
| `ready` | Ready for the next stage — agent can proceed |
| `in_progress` | Agent is actively working this stage |
| `waiting_for_human` | Agent cannot proceed without human input, approval, or a decision |
| `blocked` | External issue prevents progress — not an agent decision, not a human decision |
| `archived` | Feature is complete and has been archived |

**`waiting_for_human` vs `blocked`:**
- Use `waiting_for_human` when the agent hit something it cannot decide alone:
  approval required, MFA needed, an ambiguous requirement, a destructive action
  that needs sign-off, or any question only the human can answer.
- Use `blocked` when something external is preventing progress and neither the
  agent nor the human can fix it right now: a downstream service is down,
  a dependency is unavailable, a required environment doesn't exist yet.

---

## STATE.yaml Update Rules

Update `STATE.yaml` whenever any of the following change:

- `status` — any transition
- `stage.workflow` or `stage.owner` — when advancing to the next workflow
- `next_action` — what the next agent or human should do
- `repos` — when a worktree is created or removed
- `outputs` — when a required output is completed

Write a `history` entry for every stage transition, block, or wait event.

### History entry format

```yaml
history:
  - at: <RFC3339 timestamp>
    stage: <stage name>
    owner: <worker id or "human">
    result: <one line describing what happened>
```

---

## When to Set `waiting_for_human`

Run this command when you need the human before you can continue:

```
orc wait <ticket> "<clear description of what you need and why you stopped>"
```

Then stop. Do not guess, do not proceed, do not write placeholder content.
Include enough context in the reason that the human can act without reading the full history.

Common triggers:
- A destructive action requires explicit approval (force push, drop table, delete files)
- MFA or interactive login is required
- A requirement in TICKET.md or SPEC.md is ambiguous and the answer changes the approach
- A decision has high cost-of-mistakes and the agent is not confident
- The human needs to review and sign off before the next external action (open PR, post comment)

The human will resolve the issue and run `orc advance <ticket> --workflow <next-workflow>` to continue.

---

## When to Set `blocked`

Run this command when an external condition prevents progress:

```
orc block <ticket> "<description of what is blocked and what needs to change>"
```

Common triggers:
- A required service or environment is down
- A dependency has not been published yet
- A required access or credential has not been granted
- A merge conflict requires a human decision that affects other teams

---

## Error Handling

If any step fails in a way that prevents the stage from completing:

1. Do not leave STATE.yaml in a partially updated state
2. Set `status` to `waiting_for_human` (if you need input) or `blocked` (if external)
3. Write a clear `next_action.prompt` describing what failed and what is needed
4. Write a `history` entry recording the failure
5. Stop — do not continue to the next step or stage

---

## Starting a Stage

At the beginning of every session, before doing any work, run:

```
orc start <ticket>
```

This sets `status: in_progress` and records a history entry. If the agent session
ends unexpectedly, the status reflects that work was in progress — the next session
knows to check for partial work before continuing.

Also run `orc show <ticket> --json` to read current state:

```
orc show <ticket> --json
```

Returns structured JSON with ticket, slug, status, workflow, inputs, outputs, and
next action — easier to parse than the display output.

To see the recommended next worker and launch command as JSON:

```
orc next <ticket> --json
```

Returns ticket, status, workflow, cwd, prompt, worker id, product, model, and the
full launch command string.

---

## Advancing to the Next Workflow

When a workflow completes successfully:

1. Confirm all required outputs are written
2. Run:
   ```
   orc advance <ticket> --workflow <next-workflow> --owner <next-owner> --result "<what was accomplished>"
   ```
   This updates `stage.workflow`, `stage.owner`, sets `status: ready`, clears `next_action`,
   and writes a history entry automatically.
3. Do not hand-edit STATE.yaml for workflow transitions — use `orc advance`
