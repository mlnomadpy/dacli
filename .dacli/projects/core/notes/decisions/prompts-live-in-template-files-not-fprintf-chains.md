---
id: d-prompts-live-in-template-files-not-fprintf-chains
kind: note
note_kind: decision
created: 2026-07-21T15:35:19Z
created_by: a-root
---
# Prompts live in template files, not Fprintf chains
## Chose
Prompts live in template files, not Fprintf chains
## Rejected
prose inline in Go code
## Because
a prompt in code cannot be audited, diffed in a PR, or improved without recompiling; embedded defaults + workspace overrides make prompt tuning an attributable workspace commit
