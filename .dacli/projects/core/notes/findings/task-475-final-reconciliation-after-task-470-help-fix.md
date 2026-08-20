---
id: f-task-475-final-reconciliation-after-task-470-help-fix
kind: note
note_kind: finding
created: 2026-08-19T12:45:47Z
created_by: a-root
about: "[[475]]"
severity: major
---
# Task 475 final reconciliation after task 470 help fix
Task 470 commit 790f18f now aligns implemented github pull --dry-run and github sync preview forms with help, so revert the task 475 workaround that forbids or removes `github pull <project> --dry-run`. Restore pull-only inbound preview where appropriate. Also satisfy acceptance criterion 11 exactly: direct PR must reach observed merged plus trunk state before owner accept and github push closure; ship owns the separate wave accept-plus-integrate transaction; no dedicated runtime cooldown clear or expiry command is shipped.
