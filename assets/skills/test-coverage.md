Assess test coverage for the code at $ARGUMENTS (file path or directory).

Do not run coverage tooling — reason from reading the source and test files directly.

**What to check**

For each function, method, or component in scope, identify:
- Does a test exist for it?
- Does it cover the happy path?
- Does it cover failure / edge cases (empty input, nulls, auth failure, network error)?
- Are any branches (if/else, switch, ternary) left uncovered?

**Report format**

List each item with its coverage status:
- ✓ well covered
- ~ partially covered (note what is missing)
- ✗ not covered

Then summarise:
- Which missing tests are blocking (critical paths, security-sensitive logic)?
- Which are suggestions (minor branches, unlikely edge cases)?

If coverage looks good overall, say so clearly — the goal is signal, not exhaustive checking.
