---
id: f-accept-ignored-authoritative-merged-pr-state-for-squash-commits
kind: note
note_kind: finding
created: 2026-08-12T19:24:10Z
created_by: a-codex-maintainer-djpe71
about: "[[397]]"
severity: major
---
# accept ignored authoritative merged PR state for squash commits
internal/store/landing.go:41 previously called only ResolveBranchRefs/LandingOfRefs, while internal/features/vcs/lifecycle.go:426 queried GitHub PR state first. A GitHub squash merge replaces the original task commit, so ancestry reports unlanded even when the task PR is MERGED. Regression red line: confirmed squash-merged PR = 3, want landed without --allow-unlanded.
