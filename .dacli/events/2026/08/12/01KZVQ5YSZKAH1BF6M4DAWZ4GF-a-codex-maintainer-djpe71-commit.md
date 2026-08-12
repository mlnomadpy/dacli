---
id: 01KZVQ5YSZKAH1BF6M4DAWZ4GF
kind: event
event_kind: commit
created: 2026-08-12T19:28:55Z
created_by: a-codex-maintainer-djpe71
about: "[[t-01KZVNJ41JQ2KEDJ96NQP35FWT]]"
origin: agent
applied: true
---
16c92ad 397: recognize squash-merged task PRs

GitHub's squash merge replaces the task commit, so accept now uses the task PR's authoritative MERGED state before falling back to conservative branch ancestry. Closed-unmerged PRs and unrelated branches remain unlanded.

Red test: TestSquashMergedPRReadsAsLandedWithoutOverride failed with confirmed squash-merged PR = 3, want landed without --allow-unlanded.
role: codex-maintainer
