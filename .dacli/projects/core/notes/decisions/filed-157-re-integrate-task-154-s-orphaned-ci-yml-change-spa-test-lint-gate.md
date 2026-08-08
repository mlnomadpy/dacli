---
id: d-filed-157-re-integrate-task-154-s-orphaned-ci-yml-change-spa-test-lint-gate
kind: note
note_kind: decision
created: 2026-07-26T21:02:35Z
created_by: a-q4pq8c6yk5
about: [[084]]
---
# Filed 157: re-integrate task 154's orphaned ci.yml change (SPA test/lint gate absent on main) as the single highest-value evidence-based change
## Chose
Filed 157: re-integrate task 154's orphaned ci.yml change (SPA test/lint gate absent on main) as the single highest-value evidence-based change
## Rejected
Filing a specific shadcn-dashboard (152) component bug, re-filing the burn Rate/Ceiling population issue (already tasks 149/153), or the reviewer.md missing role_kind data backfill (already covered by 153's finding)
## Because
The 154 gap is a CURRENT, verified regression-protection hole that dominates any single component bug: 'git merge-base --is-ancestor 6e142c9 HEAD' proves 154's ci.yml change never merged, so on main today ci.yml (.github/workflows/ci.yml:24-36) runs only npm build — the 15 frontend test files (incl. the ones guarding 149 and 150) never execute in CI, so ANY frontend regression ships green. A single component-bug filing would be caught by those very tests once the gate lands, so restoring the gate strictly dominates. It is not a near-duplicate of 149/153 (burn population) nor 115 (systemic loop accept-timing) — it is the concrete instance whose remedy is a clean 6-line re-apply. reviewer.md role_kind is minor + already in 153's scope.
