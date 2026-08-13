---
id: d-enforce-the-patched-lint-toolchain-in-the-existing-workflow-contract-test
kind: note
note_kind: decision
created: 2026-08-13T22:44:50Z
created_by: a-fixer-8skqtd
about: "[[438]]"
github:
  issue: 644
  repo: mlnomadpy/dacli
---
# Enforce the patched lint toolchain in the existing workflow contract test
## Chose
Enforce the patched lint toolchain in the existing workflow contract test
## Rejected
Change only ci.yml without regression coverage
## Because
The contract test runs in CI and prevents a future edit from lowering the security-scan standard library below Go 1.25.13 while allowing later 1.25 patch releases.
