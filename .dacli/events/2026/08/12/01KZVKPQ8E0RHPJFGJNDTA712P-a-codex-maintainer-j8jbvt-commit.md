---
id: 01KZVKPQ8E0RHPJFGJNDTA712P
kind: event
event_kind: commit
created: 2026-08-12T18:28:10Z
created_by: a-codex-maintainer-j8jbvt
about: "[[t-01KZVAWWJV12EC8VW19FSCBY7Q]]"
origin: agent
applied: true
---
45c7c2d 382: keep status reads from finalizing detached runs

Status commands no longer invoke the detached-run lifecycle sweep; explicit wait remains responsible for finalization and exit recording.

Red test before fix:
TestStatusReadsDoNotFinalizeALiveRunWhenProcessIdentityIsHidden: status reads finalized a run whose guardian is alive: outcome: no visible result (detached)

Verification: focused execution package passes; gofmt and go vet pass. Full go test ./... is blocked by the pre-existing internal/cli TestE2EFixtureRepoGoesFromEmptyToShipped worker-spawn failure. golangci-lint is unavailable.
role: codex-maintainer
