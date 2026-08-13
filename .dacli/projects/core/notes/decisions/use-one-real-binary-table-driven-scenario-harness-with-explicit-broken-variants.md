---
id: d-use-one-real-binary-table-driven-scenario-harness-with-explicit-broken-variants
kind: note
note_kind: decision
created: 2026-08-13T20:00:14Z
created_by: a-codex-maintainer-grz3zz
about: "[[434]]"
github:
  issue: 616
  repo: mlnomadpy/dacli
---
# Use one real-binary table-driven scenario harness with explicit broken variants
## Chose
Use one real-binary table-driven scenario harness with explicit broken variants
## Rejected
Duplicate command-call mocks or separate hand-maintained shell suites
## Because
real temporary repositories expose composition failures, shared outcome helpers keep the command documented, and mutations prove every assertion can go red
