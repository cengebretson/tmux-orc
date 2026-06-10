# SPEC.md — MAINT-12

## Context

Oncall has been woken up 12 times in the past 30 days by alerts that either
auto-resolved or were duplicates of a known noisy condition. Alert fatigue is
reducing response confidence. This ticket triages the alert inventory and
produces a set of tuning recommendations.

## Scope

### In scope
- Inventory all active alerts in the monitoring system
- Classify each as: signal (real incidents), noise (frequent false positives), or unknown
- Identify alerts firing more than 3 times per week with no oncall action taken
- Produce a list of recommended changes: threshold adjustments, silence windows, or removals

### Out of scope
- Implementing the tuning changes (separate follow-on ticket)
- Alerting infrastructure changes (Prometheus rule files, etc.)
- New alerts

## Deliverable

`triage/alert-inventory.md` — a table of all alerts with classification, fire frequency,
last-action date, and recommendation.
