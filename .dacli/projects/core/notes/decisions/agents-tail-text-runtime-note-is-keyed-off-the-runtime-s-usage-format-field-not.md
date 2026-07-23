---
id: d-agents-tail-text-runtime-note-is-keyed-off-the-runtime-s-usage-format-field-not
kind: note
note_kind: decision
created: 2026-07-23T12:43:29Z
created_by: a-wrwzzt98sh
about: [[099]]
---
# agents --tail: text-runtime note is keyed off the runtime's usage_format field, not per-run state
## Chose
agents --tail: text-runtime note is keyed off the runtime's usage_format field, not per-run state
## Rejected
store a flag on the run record at spawn time
## Because
runtime.UsageFormat is already the single source of truth for stream-json vs text (execRuntime reads it to decide argv); looking it up live via store.LoadRuntime (cached per agents-list call) needed no new per-run state and stays correct even if the runtime adapter is edited after the run started
