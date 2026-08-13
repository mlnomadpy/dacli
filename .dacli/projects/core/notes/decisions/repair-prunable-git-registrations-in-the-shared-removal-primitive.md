---
id: d-repair-prunable-git-registrations-in-the-shared-removal-primitive
kind: note
note_kind: decision
created: 2026-08-13T23:09:55Z
created_by: a-fixer-7w7rgg
about: "[[439]]"
github:
  issue: 649
  repo: mlnomadpy/dacli
---
# Repair prunable Git registrations in the shared removal primitive
## Chose
Repair prunable Git registrations in the shared removal primitive
## Rejected
Special-case preview or duplicate eligibility in the CLI handler
## Because
preview and apply already share store.ReclaimableWorktrees; the disagreement occurs only when git worktree remove refuses a registered checkout whose .git link vanished, so repairing removal preserves one eligibility predicate for CLI and loop callers
