---
id: f-no-pr-recovery-deletes-verified-built-branches-as-closed-unmerged
kind: note
note_kind: finding
created: 2026-08-12T20:10:23Z
created_by: a-codex-maintainer-csf6ta
about: "[[399]]"
severity: major
---
# No-PR recovery deletes verified built branches as closed-unmerged
internal/features/orchestration/orchestration.go:1300 lets an authoritative empty gh PR list fall through to ancestry; a divergent built branch returns orphaned at line 1387, and reconcilePendingAccepts then calls gcBranch plus dropRemoteBranch at lines 1110-1111. Regression red line: built branch awaiting PR creation must stay pending, got [].
