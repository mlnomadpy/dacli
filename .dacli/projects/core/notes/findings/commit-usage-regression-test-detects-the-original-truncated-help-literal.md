---
id: f-commit-usage-regression-test-detects-the-original-truncated-help-literal
kind: note
note_kind: finding
created: 2026-08-19T14:09:43Z
created_by: a-fixer-gcha7z
about: "[[t-01M0D2KPCZ5PEFXJS4B0J59Z5C]]"
severity: major
---
# Commit usage regression test detects the original truncated help literal
Mutation evidence: restoring the truncated commit Usage made the focused internal/cli test fail at internal/cli/vcs_test.go:45 because help omitted the canonical contract.
