# Code Review

**Ticket:** <!-- ticket id -->  
**Reviewed by:** <!-- worker id -->  
**Date:** <!-- date -->

## Verdict

<!-- Set exactly one verdict below. The verdict drives routing — do not add custom values. -->

| Verdict | Meaning | Agent action |
|---------|---------|--------------|
| `approved` | Ready to open a PR | `orc advance --workflow pr-open` |
| `needs-changes` | Specific fixes required — see Findings | `orc advance --workflow develop` (up to 3 cycles, then escalate) |
| `blocked` | Design issue, spec conflict, or missing requirement — human decision required | `orc wait` |

**verdict: <!-- approved | needs-changes | blocked -->**

## Findings

<!-- Tag each finding. Reviewer must address all [bug], [spec], and [risk] items.
     [style] and [minor] are advisory.

     [bug]   — incorrect behavior or logic error
     [spec]  — does not match SPEC.md or TICKET.md requirements
     [risk]  — potential security, data, or reliability issue
     [style] — code quality or readability (non-blocking)
     [minor] — small nitpick, optional fix
-->

## Summary

<!-- One or two sentences on overall quality and readiness to open a PR -->
