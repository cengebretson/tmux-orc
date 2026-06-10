# SPEC.md — FEAT-001

## Context

Users working in low-light environments have reported eye strain with the
current light theme. A system-preference-respecting dark mode is the most
requested UI feature in the last two feedback cycles.

## Scope

### In scope
- Dark color palette matching system preference (`prefers-color-scheme: dark`)
- Manual toggle stored in user preferences (persists across sessions)
- All primary views: dashboard, feature list, detail panels
- Accessible contrast ratios (WCAG AA minimum)

### Out of scope
- Per-component theme customization
- High-contrast mode
- Print stylesheet

## Acceptance Criteria

- Toggle switch in user settings applies dark mode immediately without reload
- Preference persists after logout/login
- All UI components pass WCAG AA contrast check in both themes
- No flash of unstyled content on page load when dark mode is active

## Open Questions

- [ ] CSS variables vs Tailwind dark: variant? Confirm with frontend team.
- [ ] System preference auto-detection only, or also default to dark for new users?
