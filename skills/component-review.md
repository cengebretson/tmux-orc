Review the React component at $ARGUMENTS (or the most recently edited component if no path given).

Check the following and report findings grouped by severity (blocking / suggestion):

**Structure**
- Is the component doing one thing? If it has multiple responsibilities, flag it for splitting.
- Are props typed with TypeScript interfaces or types?
- Are default props handled?

**Hooks and state**
- Is state minimal — nothing that can be derived is stored?
- Are `useEffect` dependencies correct and complete?
- Are there any missing cleanup functions (subscriptions, timers, event listeners)?

**Accessibility**
- Do interactive elements have accessible labels (`aria-label`, `aria-labelledby`, or visible text)?
- Are focusable elements reachable via keyboard?
- Are images missing `alt` text?

**Performance**
- Are expensive computations wrapped in `useMemo`?
- Are callbacks passed as props wrapped in `useCallback`?
- Could any child components benefit from `React.memo`?

**Tests**
- Is there a test file for this component?
- Do tests cover the main render path and key user interactions?

End with a one-line summary: PASS, PASS WITH SUGGESTIONS, or NEEDS WORK.
