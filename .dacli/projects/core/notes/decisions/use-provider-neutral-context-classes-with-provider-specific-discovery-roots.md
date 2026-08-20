---
id: d-use-provider-neutral-context-classes-with-provider-specific-discovery-roots
kind: note
note_kind: decision
created: 2026-08-19T13:23:20Z
created_by: a-maintainer-1cw4s7
about: "[[t-01M0AEYFXAB22RE9Y2SH9WZZKR]]"
github:
  issue: 743
  repo: mlnomadpy/dacli
---
# Use provider-neutral context classes with provider-specific discovery roots
## Chose
Use provider-neutral context classes with provider-specific discovery roots
## Rejected
Add one shared ignore-user-config flag contract across all runtimes
## Because
Claude, Codex, Gemini, Copilot, and arbitrary executors discover different config, skill, extension, and MCP roots; a shared flag would recreate the false hermeticity claim from issue 691
