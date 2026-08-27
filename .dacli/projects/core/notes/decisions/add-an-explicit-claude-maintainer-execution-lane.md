---
id: d-add-an-explicit-claude-maintainer-execution-lane
kind: note
note_kind: decision
created: 2026-08-27T11:54:20Z
created_by: a-root
---
# Add an explicit Claude maintainer execution lane
## Chose
Add a separately named implementer role with the maintainer method, scope, capacity, and consequence tier, bound to cc-rw/opus. Automatic cost routing can select it with a durable explanation without vendor branching or silent substitution.
## Rejected
Wait for Codex-rw recovery or silently rewrite maintainer's runtime
## Because
Codex-rw behavioral preflight is incompatible at the sandbox layer, while cc-rw is probed compatible. A separately named role keeps provider selection visible and recorded; explicit --impl-role remains authoritative.
