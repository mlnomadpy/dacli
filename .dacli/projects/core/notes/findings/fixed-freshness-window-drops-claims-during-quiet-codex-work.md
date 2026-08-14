---
id: f-fixed-freshness-window-drops-claims-during-quiet-codex-work
kind: note
note_kind: finding
created: 2026-08-14T10:14:03Z
created_by: a-maintainer-j68p78
about: "[[t-01KZZVFWZWP3M2KX52E1FF6CMA]]"
severity: major
---
# Fixed freshness window drops claims during quiet Codex work
internal/features/execution/execution.go:2885 previously trusted transcript mtime for only 15 seconds after process identity became unobservable; the issue #672 regression test failed at claim_release_test.go:141 because liveAgents recovered the run and atomically cleared its explicit claims before the configured runtime timeout.
