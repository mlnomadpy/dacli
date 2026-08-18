---
id: d-allow-root-to-dismiss-proposals-from-retired-actors-on-recognized-loop-anchors
kind: note
note_kind: decision
created: 2026-08-18T14:31:48Z
created_by: a-fixer-rmqgbs
about: "[[t-01M0AKHSFGWWSMDFCWCE9RYCGQ]]"
---
# Allow root to dismiss proposals from retired actors on recognized loop anchors
## Chose
Allow root to dismiss proposals from retired actors on recognized loop anchors
## Rejected
Treat any task owned by loop as eligible for root recovery
## Because
Combining the synthetic owner with Task.IsLoopAnchor confines recovery to the standing anchor while ordinary and malformed targets remain fail-closed.
