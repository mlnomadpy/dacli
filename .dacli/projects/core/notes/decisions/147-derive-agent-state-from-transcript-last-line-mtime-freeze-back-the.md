---
id: d-147-derive-agent-state-from-transcript-last-line-mtime-freeze-back-the
kind: note
note_kind: decision
created: 2026-07-26T15:32:57Z
created_by: a-1dme0jhygs
about: [[147]]
---
# 147: derive agent state from transcript last-line + mtime freeze; back the transcript/diff links with two read-only dashboard endpoints
## Chose
147: derive agent state from transcript last-line + mtime freeze; back the transcript/diff links with two read-only dashboard endpoints
## Rejected
computing state from proc RAM/CPU, or faking a diff link that resolves to nothing
## Because
the transcript already distinguishes thinking (assistant text) from acting ([tool: X]); mtime freeze past a threshold is the only honest hung signal; a text runtime buffering to exit is waiting not stalled. Links must resolve to real read-only endpoints (rendered transcript; git diff HEAD of the run's worktree/checkout via shared gitx) or they lie.
