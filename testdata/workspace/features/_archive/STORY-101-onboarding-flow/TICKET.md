# TICKET.md — STORY-101

## Summary

Redesign the new user onboarding flow with a multi-step wizard.

## Description

First-time users drop off at a high rate because the current setup screen asks
for too many things at once. Replace it with a 3-step wizard: account basics,
connect a repo, invite teammates. Each step is independently saveable so users
can return and complete it later.

## Acceptance Criteria

- [x] 3-step wizard replaces the single setup page
- [x] Progress is persisted — incomplete users resume where they left off
- [x] Each step validates independently before allowing Next
- [x] Skip available on the "invite teammates" step
- [x] Onboarding completion rate tracked via analytics event

## Links

- Ticket: https://stories.example.com/STORY-101
- PR: https://github.com/example/my-app/pull/74
