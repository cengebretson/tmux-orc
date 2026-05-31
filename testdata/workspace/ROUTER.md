# ROUTER.md — test fixture workspace

## Session Root

Start every agent session at the workspace root — the directory containing this file.
Read the workspace docs here first, then navigate to the repo or worktree for code work.

---

## Repos

| Name | Path | Purpose |
|------|------|---------|
| my-app | ../my-app | Main application code, APIs, and frontend |
| infra | ../infra | Terraform and deployment configuration |
| shared-libs | ../shared-libs | Shared utility libraries used across services |

---

## Worktrees

Worktrees live inside this workspace under `worktrees/<repo-name>/<ticket-slug>`.

---

## Workflows

| Workflow       | Purpose                           |
|----------------|-----------------------------------|
| intake         | Load ticket context               |
| develop        | Feature implementation            |
| pr-open        | Open and submit a pull request    |
| pr-repair      | Fix CI failures or review feedback|
| qa-automation  | QA planning and test execution    |
