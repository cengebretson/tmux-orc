---
id: worker-id
name: Worker Display Name
product: claude
kind: agent
model: claude-sonnet-4-6
thinking: medium
cost_tier: medium
default_tmux_window: claude
launch_mode: foreground
# stages: list of stage names this worker handles (develop, code-review, etc.)
#   Omit to match any stage. Used by orc next fallback worker selection.
# workflows: list of pipeline names (default, hotfix, etc.) — reserved for
#   future pipeline-level routing; not used for stage matching.
---

# Worker Display Name

## Role

<!-- One sentence describing this worker's primary responsibility -->

## Best For

- 
- 

## Avoid

- 
- 

## Permission Boundaries

Ask before:

- Installing dependencies
- Rewriting Git history
- Writing to external systems (Jira, GitHub, CI)
- Starting background agents

## Launch Template

```bash
claude --add-dir {{workspace}} "{{prompt}}"
```

## Prompt Template

Continue {{ticket}} using:

- Feature state: `features/{{slug}}/STATE.yaml`
- Stage: `stages/{{stage}}.md`
- Current stage: `{{stage}}`
- Expected outputs: {{outputs}}

Run repo commands with cwd set to `{{cwd}}`.
