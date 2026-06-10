# PLAN.md — MAINT-12

## Approach

Pull the last 30 days of alert history from the monitoring dashboard. Cross-reference
with oncall log to identify which firings had any human action. Classify and document
each alert.

## Steps

- [ ] Export alert history from Grafana (last 30 days, all rule groups)
- [ ] Cross-reference with PagerDuty incident log — match by time window
- [ ] Build `triage/alert-inventory.md` with columns: alert, fires/week, last-action, classification
- [ ] Identify top 10 noisiest with no action — flag for removal or silence
- [ ] Write recommendations section with specific threshold changes
- [ ] Run findings past oncall rotation before closing ticket

## Classification Key

- **signal** — fires when there is a real incident, oncall acts
- **noise** — fires frequently, oncall almost never acts, or always auto-resolves
- **unknown** — insufficient data, needs more observation
