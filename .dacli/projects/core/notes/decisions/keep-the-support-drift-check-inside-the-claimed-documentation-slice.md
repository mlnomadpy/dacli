---
id: d-keep-the-support-drift-check-inside-the-claimed-documentation-slice
kind: note
note_kind: decision
created: 2026-08-12T20:15:19Z
created_by: a-codex-maintainer-p44wb5
about: "[[367]]"
github:
  issue: 539
  repo: mlnomadpy/dacli
---
# Keep the support drift check inside the claimed documentation slice
## Chose
Keep the support drift check inside the claimed documentation slice
## Rejected
Override the path-claim refusal to add an internal/mcp package test
## Because
The claim gate policy-refused internal/mcp. A docs package test can read the shipped preset and MCP registry tables and fail on count or public-claim drift without changing either feature slice.
