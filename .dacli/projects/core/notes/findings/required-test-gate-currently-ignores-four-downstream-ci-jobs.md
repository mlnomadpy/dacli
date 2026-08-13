---
id: f-required-test-gate-currently-ignores-four-downstream-ci-jobs
kind: note
note_kind: finding
created: 2026-08-13T19:20:37Z
created_by: a-codex-maintainer-xytv4d
about: "[[428]]"
severity: major
---
# Required test gate currently ignores four downstream CI jobs
.github/workflows/ci.yml:193-201 gives job test only needs: test-matrix and asserts only that result, while lint, clean-checkout, release-snapshot, and cross-compile are separate jobs; therefore those jobs can fail or cancel after the stable required test context succeeds. GitHub issue #600 could not be read because api.github.com was unreachable.
