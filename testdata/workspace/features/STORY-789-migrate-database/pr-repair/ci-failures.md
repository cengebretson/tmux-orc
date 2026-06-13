# PR Repair — STORY-789

## Failure
`staging-deploy` job cannot reach the staging cluster — environment is down for
maintenance, not a code defect.

## Action
Paused pending staging availability. Re-run `orc next` once staging is back; no
code changes required.
