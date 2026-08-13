---
id: d-use-typed-command-results-serialized-through-the-cli-process-boundary
kind: note
note_kind: decision
created: 2026-08-12T19:45:34Z
created_by: a-codex-maintainer-xscvft
about: "[[366]]"
github:
  issue: 534
  repo: mlnomadpy/dacli
---
# Use typed command results serialized through the CLI process boundary
## Chose
Use typed command results serialized through the CLI process boundary
## Rejected
add formatter-specific JSON parsing in each orchestration caller
## Because
A result carried on clikit.Ctx lets spawn and integrate expose typed facts in-process, while the app serializes them for subprocess callers without coupling control flow to stdout or feature slices.
