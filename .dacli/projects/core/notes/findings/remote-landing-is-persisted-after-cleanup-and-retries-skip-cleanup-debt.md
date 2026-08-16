---
id: f-remote-landing-is-persisted-after-cleanup-and-retries-skip-cleanup-debt
kind: note
note_kind: finding
created: 2026-08-16T18:02:49Z
created_by: a-maintainer-6w1mv4
about: "[[t-01KZYW7MC5V9B7QBMXFMAVT5VG]]"
severity: major
---
# Remote landing is persisted after cleanup and retries skip cleanup debt
internal/features/vcs/lifecycle.go:1531 performs worktree/branch cleanup before eventlog.Append at line 1573, while cmdIntegrate's recordedRemoteIntegration shortcut at line 1244 increments merged and continues without retrying cleanup. A dirty/interrupted worktree therefore leaves cleanup debt that integrate never reclaims.
