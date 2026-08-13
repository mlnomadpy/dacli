---
id: f-commit-regression-deterministically-recreates-root-m-consumption
kind: note
note_kind: finding
created: 2026-08-13T19:06:19Z
created_by: a-codex-maintainer-mjejj8
about: "[[427]]"
severity: major
---
# commit regression deterministically recreates root -m consumption
internal/cli/vcs_test.go:117 stages a claimed child file in an isolated worktree. Mutating away both new guards made the test fail because malformed commit -m requested subject exited 0 and created commit 71ce1f6 as a-root; restored implementation passes.
