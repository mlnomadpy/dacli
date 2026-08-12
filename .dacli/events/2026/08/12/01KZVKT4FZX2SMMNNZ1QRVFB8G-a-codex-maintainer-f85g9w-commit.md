---
id: 01KZVKT4FZX2SMMNNZ1QRVFB8G
kind: event
event_kind: commit
created: 2026-08-12T18:30:02Z
created_by: a-codex-maintainer-f85g9w
about: "[[t-01KZVEBZ5ATTV0C8C3B8ZSEDZ5]]"
origin: agent
applied: true
---
c09a679 391: preserve and handle restricted worker diagnostics

Red proof: discarding the fixture transcript failed TestFixtureWorkerDiagnosticReadsTranscriptBeforeCleanup with 'worker stderr was discarded: child transcript: <empty>'.

The restricted macOS git warning is filtered only in the E2E fixture; git status, other stderr, and claim enforcement remain intact.

Verification: gofmt -l .; go vet ./...; env GOCACHE=/private/tmp/dacli-go-cache go test ./...
Unverified: golangci-lint (binary unavailable).
role: codex-maintainer
