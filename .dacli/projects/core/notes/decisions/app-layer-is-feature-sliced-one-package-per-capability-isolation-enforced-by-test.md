---
id: d-app-layer-is-feature-sliced-one-package-per-capability-isolation-enforced-by-test
kind: note
note_kind: decision
created: 2026-07-21T17:17:59Z
created_by: a-root
---
# App layer is feature-sliced: one package per capability, isolation enforced by test
## Chose
App layer is feature-sliced: one package per capability, isolation enforced by test
## Rejected
numbered command files accreting in one package
## Because
chronology is not architecture; slices with a no-cross-import rule keep capabilities independently changeable, and the rule is a failing test rather than a convention
