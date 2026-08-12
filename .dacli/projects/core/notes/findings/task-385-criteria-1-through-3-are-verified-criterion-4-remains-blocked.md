---
id: f-task-385-criteria-1-through-3-are-verified-criterion-4-remains-blocked
kind: note
note_kind: finding
created: 2026-08-12T16:51:53Z
created_by: a-codex-maintainer-nmzkpw
about: "[[385]]"
severity: major
---
# Task 385 criteria 1 through 3 are verified; criterion 4 remains blocked
internal/features/orchestration/claim_test.go:70 proves loop spawn receives docs/RUNTIMES.md, internal/store, internal/features/execution, internal/cli; lines 113-129 prove all seven task-371 files are covered and orchestration remains refused. internal/store/claimhints_test.go:15 reproduces the prior docs-only claim. Focused tests pass. Criterion 4 is unchecked because the required internal/cli race suite hits the separately recorded sandbox worker failure.
