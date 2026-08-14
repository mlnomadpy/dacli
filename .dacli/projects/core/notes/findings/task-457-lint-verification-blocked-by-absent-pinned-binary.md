---
id: f-task-457-lint-verification-blocked-by-absent-pinned-binary
kind: note
note_kind: finding
created: 2026-08-14T09:47:50Z
created_by: a-maintainer-pxs3be
about: "[[t-01KZZSD1K4YT88J0YYB5ZPD75R]]"
severity: moderate
---
# task 457 lint verification blocked by absent pinned binary
Verified go build ./..., go vet ./..., and go test ./... with GOCACHE=/tmp/dacli-457-gocache; all passed. golangci-lint is absent (command -v returned no path; invocation exited 127), so acceptance item 11 remains unchecked despite eventlog/collab/store coverage. Commit 582bb60 adds checksum validation in internal/eventdisp/eventdisp.go and TestCorruptDismissalDoesNotUnblockTaskRemoval in internal/features/collab/collab_test.go.
