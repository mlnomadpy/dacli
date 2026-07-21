---
id: r-retro-001
kind: note
note_kind: ref
created: 2026-07-21T17:31:50Z
created_by: a-root
about: [[t-01KY2J1BRB87WQHQMS9RVA5XCY]]
scope: workspace
---
# Retro: 001
## Went well
- gate predicates reused the SPM ambiguity engine unchanged — the filled check was nearly free

## Didn't go well
- the spec's nested-YAML manifest was unimplementable in the flat parser; sectioned markdown was the honest v1

## Improve next time
- when a spec prescribes a format, check it against the parser doctrine before committing to it

