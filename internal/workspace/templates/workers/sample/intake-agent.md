---
id: intake-agent
name: Intake Agent
product: claude
model: claude-sonnet-4-6
cost_tier: medium
workflows:
  - intake
stages:
  - intake
launch_mode: foreground
---

# Intake Agent

Fetches ticket context from the source system and populates the feature folder.

Read `workflows/intake/WORKFLOW.md` for source system instructions before starting.
If the ticket cannot be found, run `orc wait <ticket> "<reason>"` and stop.
