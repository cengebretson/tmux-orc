# Workflow: intake

> Before starting: read `workflows/REQUIREMENTS.md` for state update rules and error handling.

## Purpose

Load ticket context and create the feature folder. This is the required first
workflow for any ticket-driven work — nothing downstream runs until intake completes.

## Trigger

```
orc work <ticket>
```

## Source System

<!-- TODO: Update this section to match your ticketing system. -->

**Option A — Jira**
Use the `jira` MCP server. Fetch the ticket by ID and read the summary,
description, and acceptance criteria fields.

**Option B — GitHub Issues**
Use the `github` MCP server. Fetch the issue by number and read the title,
body, and any linked acceptance criteria.

**Option C — Local files**
Tickets are markdown files in `tickets/<ticket-id>.md`. Read that file directly.

**Option D — Manual**
No ticketing system. The human fills in `TICKET.md` by hand before running
`orc next <ticket>`.

<!-- Delete the options above that don't apply and keep only the one you use. -->

## Stages

### intake

**Owner:** intake agent  
**Inputs:** Ticket ID  
**Outputs:** `TICKET.md`, `SPEC.md`, `PLAN.md`

Steps:

1. Fetch the ticket from the source system defined above.
2. If the ticket cannot be found, set `STATE.yaml` `status: error` with a clear
   explanation in `next_action.prompt` and stop.
3. Populate `TICKET.md` with the ticket summary, description, and acceptance criteria.
4. Draft `SPEC.md` with context, scope, and open questions.
5. Draft `PLAN.md` with an initial approach and steps.
6. Update `STATE.yaml`: set `status: ready` and route to the next workflow and worker.

## Exit Criteria

`TICKET.md`, `SPEC.md`, and `PLAN.md` are populated. `STATE.yaml` has
`status: ready` pointing to the next workflow and worker.

## Error Handling

If the ticket cannot be found or fetched:
- Set `status: error` in `STATE.yaml`
- Set `next_action.prompt` to a clear description of what failed and what to check
- Do not populate files with placeholder content
- Stop — a human must resolve the error before work continues
