---
id: f-pr-integration-downgraded-github-outages-into-unchecked-local-landing
kind: note
note_kind: finding
created: 2026-08-14T00:24:18Z
created_by: a-maintainer-2vktb5
about: "[[449]]"
severity: major
---
# PR integration downgraded GitHub outages into unchecked local landing
internal/features/vcs/lifecycle.go:1370 previously treated push, PR-open, checks, and merge transport failures as permission to call mergeTask locally, bypassing the project's PR checks/review gate. TestIntegratePRFailsClosedOnPushNetworkError now proves main does not advance after a failed push.
