---
id: d-refactor-dashboard-sections-onto-shadcn-vue-primitives-tailwind-theme-tokens
kind: note
note_kind: decision
created: 2026-07-26T17:28:32Z
created_by: a-tja4fdtr3z
about: [[152]]
---
# Refactor dashboard sections onto shadcn-vue primitives + Tailwind theme tokens while preserving test class hooks
## Chose
Refactor dashboard sections onto shadcn-vue primitives + Tailwind theme tokens while preserving test class hooks
## Rejected
Rewrite DOM structure freely and rewrite every test to match
## Because
Tests encode real behavioral contracts (freshness dot buckets, burn alert yell, DAG critical-adjacency, board chip cap, section states). Keeping semantic marker classes (.dot/.badge/.chip/.bar/.node/.edge/.cp-chip/button.retry/[role=group]) as test hooks while restyling via Tailwind utilities + shadcn Card/Table/Badge/Button/Progress lets me swap the styling layer with near-zero test churn and no feature regression; statusColor() stays var(--open..done) per its unit test, so tokens.css keeps only status vars + pulse/mono/body scaffolding.
