---
id: d-extracted-deriveagentstate-into-new-internal-agentstate-package-extended-with
kind: note
note_kind: decision
created: 2026-08-05T13:52:10Z
created_by: a-fixer-015xkz
about: "[[270]]"
---
# Extracted deriveAgentState into new internal/agentstate package, extended with blocked and silent
## Chose
Extracted deriveAgentState into new internal/agentstate package, extended with blocked and silent
## Rejected
A second execution-only reimplementation of blocked/silent duplicating dashboard logic a third time
## Because
arch_test.go forbids execution to dashboard imports, and dashboard.go already carried three near-verbatim copies of this logic duplicated from execution.go with comments admitting it. Task 270 asks for shared derivation, not a fourth copy. agentstate is a peer to store, store already imports procmon, store cannot import agentstate back, so no cycle, and both feature slices now import it without violating the arch test since agentstate is not under internal features.
