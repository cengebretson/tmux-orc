# features/

Each subfolder is a durable context pack for one ticket. State survives session
changes, agent switches, and restarts — everything needed to pick up where work
left off is in these files.

Start a new ticket with:

```
orc work TICKET-0000
```

## Structure

```
features/
  _template/            copied for each new ticket by orc work
  _archive/             completed features moved here by orc archive
  TICKET-0000-slug/
    STATE.yaml          current stage, owner, status, next action
    TICKET.md           ticket description and acceptance criteria
    SPEC.md             context, scope, and open questions
    PLAN.md             implementation approach and steps
    WORKLOG.md          running session log
    DECISIONS.md        significant decisions and rationale
    impl/
      PR.md             PR URL and status
      QA_HANDOFF.md     implementation summary for the QA agent
    qa/
      SOURCE_CONTEXT.md repo context for the QA agent
      QA_PLAN.md        test cases and coverage plan
      RUNS.md           test run history
      QA_RESULT.md      final result and evidence
```
