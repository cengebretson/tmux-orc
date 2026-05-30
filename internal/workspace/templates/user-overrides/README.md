# user-overrides/

Local preferences for this machine. This directory is gitignored — never committed.

Use it to extend shared workspace rules with preferences that are specific to your
local environment, without affecting other contributors.

```
user-overrides/
  RULES.md    ← local additions to approval or cost rules
  TOOLS.md    ← local tool paths, aliases, or MCP server overrides
  AGENTS.md   ← local agent preferences or model overrides
```

Agents check this directory after reading shared workspace files (see `AGENTS.md`).
Overrides may add local preferences but may NOT weaken workspace-level security,
approval, secrets, compliance, or audit rules.
