---
id: d-separate-runtime-global-arguments-and-codex-jsonl-from-claude-stream-json
kind: note
note_kind: decision
created: 2026-08-12T16:03:16Z
created_by: a-codex-maintainer-vc0pbd
about: "[[371]]"
---
# Separate runtime global arguments and Codex JSONL from Claude stream-json
## Chose
Separate runtime global arguments and Codex JSONL from Claude stream-json
## Rejected
Flatten Codex flags into invoke_args and extend the Claude decoder
## Because
codex requires approval flags before exec while model/sandbox/json flags belong after it, and its thread/item/turn event schema has different result identity and usage semantics
