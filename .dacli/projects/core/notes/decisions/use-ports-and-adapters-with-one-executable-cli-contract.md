---
id: d-use-ports-and-adapters-with-one-executable-cli-contract
kind: note
note_kind: decision
created: 2026-08-13T09:37:25Z
created_by: a-root
about: "[[404]]"
github:
  issue: 556
  repo: mlnomadpy/dacli
---
# Use ports and adapters with one executable CLI contract
## Chose
Use ports and adapters with one executable CLI contract
## Rejected
Provider-specific spawn branches and independent hand-written tests
## Because
Every coding CLI must prove the same prompt, model, result, usage, timeout, cancellation, and sandbox behaviors; a shared contract keeps adapters thin and makes first-class support a testable claim.
