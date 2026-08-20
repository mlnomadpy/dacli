---
id: d-validate-canonical-playbook-commands-against-exact-help-signatures
kind: note
note_kind: decision
created: 2026-08-19T12:13:06Z
created_by: a-maintainer-ebqr9f
about: "[[t-01M0CX031NDQ5PQ8VRX1PQNWXE]]"
github:
  issue: 732
  repo: mlnomadpy/dacli
---
# Validate canonical playbook commands against exact help signatures
## Chose
Validate canonical playbook commands against exact help signatures
## Rejected
Only assert that similarly named command paths exist in source
## Because
A path-only test stayed green while next and push advertised unrelated commands; exact Usage contracts make stale documented forms fail CI.
