# PLAN.md — FEAT-001

## Approach

Use CSS custom properties for all color tokens so the dark palette is a single
override block. Store the user preference in `user_settings` and expose it via
the existing preferences API. Detect system preference on first load; manual
toggle overrides.

## Steps

- [ ] Define dark palette tokens in `styles/themes/dark.css`
- [ ] Refactor light theme to use CSS variables (audit existing hardcoded colors)
- [ ] Add `theme` field to `user_settings` table (migration)
- [ ] Expose GET/PUT /users/me/settings for theme preference
- [ ] Frontend toggle component wired to preferences API
- [ ] Server-side render initial theme class to avoid FOUC
- [ ] Accessibility audit: check contrast ratios for all interactive elements

## Risk

- Existing hardcoded colors in component stylesheets need audit before the token
  refactor — could be broader than expected. Budget 2–3 days for the audit pass.
