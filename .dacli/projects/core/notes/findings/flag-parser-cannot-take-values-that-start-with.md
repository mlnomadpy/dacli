---
id: f-flag-parser-cannot-take-values-that-start-with
kind: note
note_kind: finding
created: 2026-07-21T15:06:17Z
created_by: a-root
severity: moderate
---
# flag parser cannot take values that start with --
internal/cli/commands.go parseFlags treats any --token as a key, so '--sandbox-ro-arg "--allowedTools"' silently became 'true' and run 01KY2K8N4C sent garbage argv to claude. Support '--key=--value' explicitly or an end-of-flags marker.
