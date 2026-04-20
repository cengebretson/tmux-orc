Audit the dependencies in $ARGUMENTS (package.json, requirements.txt, Gemfile, go.mod, etc. — or the project root if not specified).

**What to check**

Read the dependency file(s) and reason about each item:

1. **Known vulnerabilities** — flag any packages you recognise as having known CVEs or security advisories. Note that you cannot run live vulnerability databases, so flag packages you have knowledge of and recommend running `npm audit` / `pip audit` / `bundle audit` for a definitive scan.

2. **Outdated majors** — flag packages pinned to a major version you know has been superseded (e.g. React 17 when 18/19 is current), especially if the older version has known issues.

3. **Abandoned packages** — flag any packages you know to be unmaintained or deprecated with a recommended replacement.

4. **Suspicious packages** — flag anything with an unusual name, low usage, or that looks like a typosquat of a well-known package.

5. **Overly broad version ranges** — flag `*` or very wide ranges (`>=1.0.0`) in production dependencies.

**Report format**
- List findings by severity: critical, high, medium, low
- For each finding: package name, issue, recommended action
- End with: run `<audit command>` for a live vulnerability scan against current advisories
