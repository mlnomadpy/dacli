---
id: d-persist-the-measured-trunk-marker-with-governor-counters-and-clear-only-the
kind: note
note_kind: decision
created: 2026-08-12T15:30:56Z
created_by: a-codex-maintainer-2amnk2
about: "[[379]]"
---
# Persist the measured trunk marker with governor counters and clear only the streak on observed advancement
## Chose
Persist the measured trunk marker with governor counters and clear only the streak on observed advancement
## Rejected
Delete or reset the whole governor snapshot automatically when trunk moves
## Because
Trunk advancement is evidence that clears the no-progress streak, but it must not erase cumulative cycles or token-window spend; an unchanged marker must preserve the prior halt.
