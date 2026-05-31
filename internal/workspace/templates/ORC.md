# ORC.md — Agent State Contract

Read this file at the start of every session.

Also read:
- `RULES.md` — what requires human approval before acting
- `AGENTS.md` — routing, tool policy, and repo commands

---

## Session Protocol

**Start every session:**
```
orc start <ticket>
orc show <ticket> --json
```
Read `stages/<stage>.md` for the current stage instructions.

**End every session with exactly one of:**
```
orc advance <ticket> --owner <next-owner> --result "<what was done>"   # stage complete
orc wait <ticket> "<what you need from the human>"                     # need input/approval
orc block <ticket> "<what is blocking progress>"                       # external blocker
```
Never end a session without updating state. Never hand-edit STATE.yaml directly.

---

## Status Values

| Status | Meaning |
|--------|---------|
| `pending` | Feature created, intake not yet run |
| `ready` | Ready for the next stage |
| `in_progress` | Agent is actively working |
| `waiting_for_human` | Needs input, approval, or a decision only a human can make |
| `blocked` | External condition prevents progress (service down, access missing, etc.) |
| `archived` | Complete |

Use `waiting_for_human` when *you* cannot proceed. Use `blocked` when *nothing* can proceed until an external condition changes.

---

## STATE.yaml Update Rules

Write a history entry for every stage transition, block, or wait:

```yaml
- at: <RFC3339>
  stage: <stage name>
  owner: <worker id or "human">
  result: <one line>
```

Also update `stage.name`, `stage.owner`, `next_action`, and `repos` whenever those change.

---

## Feature Folder

Every ticket has a context pack at `features/<ticket-slug>/`:

| File | Purpose |
|------|---------|
| `STATE.yaml` | Durable state — status, stage, owner, next action, history |
| `TICKET.md` | Ticket description and acceptance criteria |
| `SPEC.md` | Context, scope, constraints, open questions |
| `PLAN.md` | Implementation approach and steps |
| `DECISIONS.md` | Non-obvious choices — what, why, alternatives rejected |

Read `STATE.yaml` and `TICKET.md` at the start of every session. Read `SPEC.md` and `PLAN.md` before any implementation work.

---

## Stage Handoff

The feature folder is the handoff medium between stages. Read previous stage outputs before starting work. If a required input is missing, `orc wait` — do not proceed.

| Path | Written by | Read by |
|------|-----------|---------|
| `TICKET.md` | intake | all stages |
| `SPEC.md` | intake | develop, code-review |
| `PLAN.md` | intake | develop |
| `DECISIONS.md` | any stage | any stage |
| `impl/QA_HANDOFF.md` | develop | qa-automation |
| `impl/PR.md` | pr-open | qa-automation, human |
| `impl/REVIEW.md` | code-review | develop |
| `qa/QA_PLAN.md` | qa-automation | qa-automation (next session) |
| `qa/RUNS.md` | qa-automation | qa-automation, human |
| `qa/QA_RESULT.md` | qa-automation | human, archive |

---

## Recording Decisions

When you make a non-obvious choice, write it to `features/<ticket-slug>/DECISIONS.md` at the moment of the decision:

```
## <short title>
**Decision:** <what>
**Reason:** <why — constraints, tradeoffs, context>
**Alternatives:** <what else was considered and why rejected>
```

One entry per decision. Do not batch at end of session.
