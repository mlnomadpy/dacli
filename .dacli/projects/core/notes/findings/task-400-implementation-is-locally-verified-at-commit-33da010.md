---
id: f-task-400-implementation-is-locally-verified-at-commit-33da010
kind: note
note_kind: finding
created: 2026-08-12T20:22:39Z
created_by: a-codex-maintainer-w6vv23
about: "[[400]]"
severity: major
---
# Task 400 implementation is locally verified at commit 33da010
TestWaitKeepsFreshCodexRunsLiveDuringRegistrationStartup reproduces runs 01KZVSF64J/01KZVSFBXR at 12s/7s and proves agents, runs list, and wait retain both. TestRunLifecycleLivenessBoundsTranscriptActivity proves fresh transcript retention beyond the 30s startup grace and expiry after 15s inactivity. TestWaitFinalizesGoneDetachedRuns proves a stale dead launch finalizes. gofmt, vet, and go test ./... passed; lint is separately reported unavailable.
