---
id: r-acceptance-contract-for-task-520-canonical-pr-generation
kind: note
note_kind: ref
created: 2026-08-28T12:47:11Z
created_by: a-root
about: "[[520]]"
---
# Acceptance contract for task 520 canonical PR generation
Regression: one historical merged PR plus a newer open PR on the canonical task head must select the newer PR; historical merged evidence proves landing only for the current delivery generation; do not prune while current unlanded PR exists; mutation removing generation-aware selection must fail the targeted test; run focused and repository-wide Go gates.
