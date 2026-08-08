---
id: f-082-complete-on-branch-dacli-082-pr-43-gh-unknown-owner-type-now-yields-an
kind: note
note_kind: finding
created: 2026-07-22T22:14:41Z
created_by: a-aztk8559eb
about: [[082]]
severity: moderate
---
# 082 complete on branch dacli/082 (PR #43) — gh 'unknown owner type' now yields an actionable missing-project-scope error
Commit 4922dbd by a-aztk8559eb. Both acceptance criteria met: (1) dacli github project detects gh's opaque 'unknown owner type' failure (the signal a token lacks the 'project' scope Projects v2 needs, separate from repo) and tells the operator to run 'gh auth refresh -s project' instead of surfacing gh's cryptic message — every gh project subcommand routes through ghProjectCmd (internal/features/ghmirror/project.go:359), which on error tests missingProjectScope(out) (case-insensitive substring 'unknown owner type', project.go:352) and, when matched, returns fmt.Errorf with projectScopeHint (project.go:340). Wrapped call sites: item-list, list, create, field-list, field-create, item-add, item-edit. (2) committed by an agent + opened as PR #43; go build ./... exits 0 and go test ./internal/... is fully green (incl. new TestMissingProjectScope in project_test.go covering case-insensitive positives and non-scope-failure negatives). NOTE: box-checking refused for non-owner (only a-root) — owner should verify and close via dacli task check/done + dacli merge --task 082.
