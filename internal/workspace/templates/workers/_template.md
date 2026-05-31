---
id: worker-id
name: Worker Display Name
engine: claude
kind: agent
model: claude-sonnet-4-6
args:
  effort: medium
default_tmux_window: claude
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
