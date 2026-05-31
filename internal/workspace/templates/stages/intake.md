# Stage: intake

> Before starting: read `ORC.md` for state update rules and error handling.

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

## Steps

**Owner:** intake agent  
**Inputs:** Ticket ID  
**Outputs:** `TICKET.md`, `SPEC.md`, `PLAN.md`

1. Fetch the ticket from the source system defined above.
2. If the ticket cannot be found, run `orc wait <ticket> "<explanation>"` and stop.
3. Populate `TICKET.md` with the ticket summary, description, and acceptance criteria.
4. Draft `SPEC.md` with context, scope, and open questions.
5. Draft `PLAN.md` with an initial approach and steps.

## Exit Criteria

`TICKET.md`, `SPEC.md`, and `PLAN.md` are populated.

When done, run:
```
orc advance <ticket> --workflow develop --owner <worker-id> --result "Intake complete"
```

## Error Handling

If the ticket cannot be found or fetched:
- Run `orc wait <ticket> "<description of what failed and what to check>"`
- Do not populate files with placeholder content
- Stop — a human must resolve the issue before work continues
