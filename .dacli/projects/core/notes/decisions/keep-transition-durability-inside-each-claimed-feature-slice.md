---
id: d-keep-transition-durability-inside-each-claimed-feature-slice
kind: note
note_kind: decision
created: 2026-08-13T19:57:23Z
created_by: a-codex-maintainer-2j651b
about: "[[431]]"
github:
  issue: 618
  repo: mlnomadpy/dacli
---
# Keep transition durability inside each claimed feature slice
## Chose
Keep transition durability inside each claimed feature slice
## Rejected
Change shared store, model, workspace, or eventlog APIs
## Because
The live claim is limited to internal/features/queues and internal/features/stagegate; command-boundary receipts can enforce stable replay semantics and append attributed journal events without violating slice isolation or expanding the claim
