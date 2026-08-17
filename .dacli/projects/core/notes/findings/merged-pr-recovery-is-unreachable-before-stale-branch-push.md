---
id: f-merged-pr-recovery-is-unreachable-before-stale-branch-push
kind: note
note_kind: finding
created: 2026-08-17T16:08:33Z
created_by: a-maintainer-w1qy51
about: "[[t-01KZYW7MC5V9B7QBMXFMAVT5VG]]"
severity: major
---
# Merged PR recovery is unreachable before stale branch push
internal/features/vcs/lifecycle.go:1404 pushes BranchFor(t) before mergedPR is first consulted near the post-merge path; the live-shaped TestIntegratePRRecoversMergedDeletedRemoteBranchBeforePush failed with push invoked and integrated 0 branch(es).
