Check the UI at $ARGUMENTS (file path, component name, or route) for accessibility issues.

Work through each category and list any findings with file and line number:

**Keyboard navigation**
- Can all interactive elements (buttons, links, inputs, modals) be reached and activated via keyboard alone?
- Is focus order logical — does it follow the visual layout?
- Are focus styles visible (not `outline: none` without a replacement)?
- Do modals trap focus correctly and restore it on close?

**Screen reader support**
- Do all images have descriptive `alt` text (or `alt=""` if decorative)?
- Do form inputs have associated `<label>` elements or `aria-label`?
- Are icon-only buttons labelled with `aria-label`?
- Are dynamic content changes announced via `aria-live` where appropriate?

**Colour and contrast**
- Flag any text that is likely to fail WCAG AA contrast (4.5:1 for normal text, 3:1 for large text) — note you cannot measure exact values without tooling, so flag anything that looks marginal.
- Is colour used as the only means of conveying information?

**Semantics**
- Are headings in a logical hierarchy (no skipped levels)?
- Are lists marked up as `<ul>` / `<ol>` rather than divs?
- Are landmark regions present (`<main>`, `<nav>`, `<header>`, `<footer>`)?

End with a summary count: X blocking issues, Y suggestions.
