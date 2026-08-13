---
id: d-use-provider-specific-structured-formats-and-exact-help-contracts
kind: note
note_kind: decision
created: 2026-08-13T10:16:16Z
created_by: a-codex-maintainer-1d99qt
about: "[[405]]"
---
# Use provider-specific structured formats and exact help contracts
## Chose
Use provider-specific structured formats and exact help contracts
## Rejected
Treat Gemini and Copilot output and safety switches as Claude stream-json aliases
## Because
Gemini's installed schema and flags already differ; doctor must validate the exact preset flag vocabulary and only cache verified RO after all required switches are advertised.
