# TICKET.md — FEAT-001

## Summary

Add dark mode support to the web UI.

## Description

Users have requested a system-aware dark mode. The UI should respect the OS
`prefers-color-scheme` media query and persist the user's manual override in
local storage.

## Acceptance Criteria

- [ ] Dark and light themes defined with CSS custom properties
- [ ] Theme toggles correctly on first load from system preference
- [ ] Manual override persists across page refreshes
- [ ] All existing UI components pass visual regression tests in both modes

## Links

- Ticket: https://stories.example.com/FEAT-001
- PR: (pending)
