---
id: f-public-worktree-verification-regression-binds-execution-and-evidence
kind: note
note_kind: finding
created: 2026-08-19T13:41:19Z
created_by: a-fixer-4ktgam
about: "[[t-01M0AGCX64GETGKKBBJK5SKG7D]]"
severity: major
---
# Public worktree verification regression binds execution and evidence
internal/cli/verification_worktree_test.go:17 invokes task check and accept from a linked branch, asserts pwd-created probes in that checkout, exact branch/SHA evidence in the shared root record, and mutation to planning.go:472 using w.Root failed at checked-from.txt.
