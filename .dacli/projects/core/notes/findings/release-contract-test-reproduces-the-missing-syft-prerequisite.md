---
id: f-release-contract-test-reproduces-the-missing-syft-prerequisite
kind: note
note_kind: finding
created: 2026-08-13T19:43:04Z
created_by: a-codex-maintainer-9gwn2s
about: "[[429]]"
severity: major
---
# Release contract test reproduces the missing Syft prerequisite
.github/workflows/contract_test.go:47 fails on the pre-fix release.yml with: release workflow must install a pinned Syft distribution; the test also compares step positions so moving Syft after GoReleaser fails.
