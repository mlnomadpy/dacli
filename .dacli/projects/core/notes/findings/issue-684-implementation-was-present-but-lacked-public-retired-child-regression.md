---
id: f-issue-684-implementation-was-present-but-lacked-public-retired-child-regression
kind: note
note_kind: finding
created: 2026-08-18T14:40:36Z
created_by: a-maintainer-w5nkdg
about: "[[t-01M088WV1WEBW031R2046WVZSW]]"
severity: moderate
---
# Issue 684 implementation was present but lacked public retired-child regression
main already contains authorizeTaskRemoval and focused handler tests from commit 3aa18f1, but no internal/cli public-command test spawned, retired, and reconciled a child-owned duplicate; added coverage in internal/cli/agents_run_test.go.
