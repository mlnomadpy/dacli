---
id: f-global-usage-invariant-exposed-three-copied-command-paths-beyond-skillforge
kind: note
note_kind: finding
created: 2026-08-19T12:23:01Z
created_by: a-maintainer-mqc389
about: "[[t-01M0AF65RDNBEX2SEF9JC5RTMZ]]"
severity: major
---
# Global usage invariant exposed three copied command paths beyond skillforge
After adding a command-path prefix assertion, go test ./internal/cli -run TestCommandUsageMatchesHandlerUsage failed for next, shortcut add, and template show. Their tables advertised queue next, queue add, and skill show respectively at internal/features/insight/insight.go:28, internal/features/shortcuts/shortcuts.go:24, and internal/features/stagegate/stagegate.go:17; aligned them with their handlers.
