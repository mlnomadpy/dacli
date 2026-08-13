---
id: 01KZXAD8SYXM95D81F33RQ028N
kind: event
event_kind: commit
created: 2026-08-13T10:24:12Z
created_by: a-codex-maintainer-1d99qt
about: "[[t-01KZX7PXM8TKXJQJXVWZBEMC0H]]"
origin: agent
applied: true
---
76d8bc7 405: add Gemini and Copilot runtime adapters

Red tests before implementation:
- TestGeminiAndCopilotPresetsAreLeastPrivilege: missing preset gemini (and gemini-rw/copilot/copilot-rw)
- TestRuntimeDoctorVendorPresetFlagDriftFailsClosed: valid help was not verified
- TestTeeGeminiStreamJSONRecordsStructuredResult: usage input/output tokens were 0

Adds least-privilege RO/RW presets, vendor-specific structured parsing and
fail-closed installed-flag probes. Documents external authentication without
credential persistence.

Verification: gofmt -l .; go vet ./...; go test ./...
Unverified: golangci-lint (binary unavailable); Linux fixture run
role: codex-maintainer
