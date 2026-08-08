---
id: d-filed-task-154-ci-never-runs-the-spa-vitest-suite-as-the-single-highest-value
kind: note
note_kind: decision
created: 2026-07-26T17:22:26Z
created_by: a-hxr220kqc4
about: [[084]]
---
# Filed task 154 (CI never runs the SPA vitest suite) as the single highest-value evidence-based change
## Chose
Filed task 154 (CI never runs the SPA vitest suite) as the single highest-value evidence-based change
## Rejected
Filing that burn.windows (BurnWindow[]) is computed server-side, typed in types.ts, and in App.test.ts fixtures but never rendered by any component (dead payload); or re-filing the byRef %03d cross-project key collision in graph.go:114; or 'DependencyGraph.vue has no test'
## Because
The burn.windows gap is a missing-feature/dead-data issue (moderate: the server does work each 2s poll that the UI discards, but nothing is wrong on screen). The byRef collision is unreachable from the SPA (already dismissed in the task-150 decision — /api/state scopes graph per-project). 'no DependencyGraph test' is false — DependencyGraph.test.ts exists. The CI gap is the class-level defect that makes ALL of those latent: the entire frontend regression suite (14 files, incl. the tests that lock task-149 and task-150's shipped correctness fixes) has zero CI protection, so any behavioral regression ships green. It is evidence-grounded (ci.yml:24-34 vs package.json:9,13,15 + task 135:15), cheap to fix, and squarely in frontend-reviewer scope.
