---
id: f-reopen-state-had-no-durable-generation-shared-with-landing-recovery
kind: note
note_kind: finding
created: 2026-08-17T15:37:34Z
created_by: a-maintainer-z795wm
about: "[[t-01M05Y78XTYNQ4GP3AV396JFD6]]"
severity: major
---
# Reopen state had no durable generation shared with landing recovery
internal/store/remove.go previously only cleared boxes and appended prose, while internal/features/orchestration/orchestration.go keyed pending recovery by sequence/branch. The same reused branch therefore made an earlier merge indistinguishable from corrective work; internal/store/worktrees.go also classified that open reopened checkout solely by merged ancestry.
