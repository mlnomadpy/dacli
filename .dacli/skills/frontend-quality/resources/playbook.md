# Frontend quality playbook

Start from the server snapshot contract and the existing design specification.
For each surface define loading, empty, error, stale/reconnecting, and live
behavior. Do not invent write actions in a read-only dashboard.

Use semantic elements before ARIA, preserve keyboard order and visible focus,
test contrast, and state responsive collapse rules. Reuse shared tokens,
components, stores, and composables rather than creating one-off systems.

Profile before optimizing. Watch polling cadence, request overlap, reactive
fan-out, unbounded histories/lists, chart recomputation, and bundle growth.
Verify with unit tests, TypeScript checking, lint, production build, and a
manual narrow/wide viewport pass when layout changes.
