---
id: d-filed-task-125-remaining-hand-rolled-inline-list-writers-as-the-single-highest
kind: note
note_kind: decision
created: 2026-07-23T18:53:13Z
created_by: a-k6yvk61byc
about: [[084]]
---
# Filed task 125 (remaining hand-rolled inline-list writers) as the single highest-value evidence-based change
## Chose
Filed task 125 (remaining hand-rolled inline-list writers) as the single highest-value evidence-based change
## Rejected
The other recent findings (mdstore SetList 119, runtimefiles quoting 111, unbounded git/gh subprocesses 110, loop governor bugs 114-117) — each already has an open/done task or landed PR
## Because
The sibling finding f-3-more-unquoted-join-inline-list-writers (verified against source: store.go:297, shortcutfiles.go:39/42, roles.go:40 all use '['+strings.Join(v,", ")+']' while mdstore GetList uses quote-aware splitTop) documents a real write/read asymmetry that 111 and 119 explicitly left out of scope, and NO open task covers those three files — so it is concrete, evidence-grounded, unowned, and a direct completion of the round-trip-correctness thread 111/119 started
