---
id: f-task-428-first-three-criteria-are-observed-but-acceptance-remains-owner-gated
kind: note
note_kind: finding
created: 2026-08-13T19:25:12Z
created_by: a-codex-maintainer-xytv4d
about: "[[428]]"
severity: major
---
# Task 428 first three criteria are observed but acceptance remains owner-gated
.github/workflows/ci.yml makes stable job test depend on and assert success for test-matrix, lint, clean-checkout, release-snapshot, and cross-compile. .github/workflows/contract_test.go exercises failure and cancellation for every dependency and is run explicitly by the test matrix. task check 428 was refused because only owner a-root may check acceptance. Criterion 4 remains unverified because GitHub is unreachable.
