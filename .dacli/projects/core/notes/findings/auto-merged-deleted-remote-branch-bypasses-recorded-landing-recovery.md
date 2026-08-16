---
id: f-auto-merged-deleted-remote-branch-bypasses-recorded-landing-recovery
kind: note
note_kind: finding
created: 2026-08-16T18:32:17Z
created_by: a-root
about: "[[t-01KZYW7MC5V9B7QBMXFMAVT5VG]]"
severity: major
---
# Auto-merged deleted remote branch bypasses recorded landing recovery
Live proof after PR #678 merged at 27acc22: from clean current main, dacli integrate --pr --tasks 452 --into main pushed the stale attached local task branch, then gh pr create failed with GraphQL No commits between main and dacli/452.... No Integrated via PR event existed because auto-merge occurred through dacli pr, so the recordedRemoteIntegration shortcut and cleanup retry were never reached. Required fix: discover a merged PR before pushing/creating, persist its merge identity, then run retryable cleanup without recreating the remote branch.
