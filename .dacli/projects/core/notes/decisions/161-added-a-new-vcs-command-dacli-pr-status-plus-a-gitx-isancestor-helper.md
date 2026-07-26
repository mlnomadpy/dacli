---
id: d-161-added-a-new-vcs-command-dacli-pr-status-plus-a-gitx-isancestor-helper
kind: note
note_kind: decision
created: 2026-07-26T22:23:01Z
created_by: a-4k8g38rpse
about: [[161]]
---
# 161: added a new vcs command (dacli pr status) plus a gitx.IsAncestor helper, instead of folding a landed-check into dacli doctor
## Chose
161: added a new vcs command (dacli pr status) plus a gitx.IsAncestor helper, instead of folding a landed-check into dacli doctor
## Rejected
Adding an 'orphaned-branch' detector to insight.cmdDoctor next to the existing owner-liveness orphan check
## Because
doctor lives in the insight feature slice and TestFeatureSlicesAreIsolated (internal/cli/arch_test.go) forbids a feature slice from importing another feature slice's package; the gh PR-state check needs vcs's runGH (and its network-error/mocking conventions), so the check has to live in vcs itself. Exposing it as its own 'dacli pr status <task>' command (matching the existing 'worktree add'/'pr' path convention) keeps it usable both by an interactive operator and by a reviewer-role agent mid-audit, without a cross-slice import.
