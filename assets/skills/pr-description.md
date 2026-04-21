Generate a pull request description for the current branch.

First, gather context:
```bash
git log main..HEAD --oneline          # commits on this branch
git diff main..HEAD --stat            # files changed
git diff main..HEAD                   # full diff
```

Then write a PR description with these sections:

## What
A concise summary of what this PR changes. One to three sentences.

## Why
The motivation — what problem does this solve, or what feature does it add? Reference any related issue numbers if they appear in commit messages or branch names.

## Changes
A bullet list of the notable changes grouped logically (not a file-by-file list):
- New: ...
- Modified: ...
- Removed: ...

## How to test
Step-by-step instructions a reviewer can follow to verify the change works correctly. Include any setup steps, commands to run, or expected outcomes.

---

Output the description as markdown ready to paste into `gh pr create --body "..."`.
