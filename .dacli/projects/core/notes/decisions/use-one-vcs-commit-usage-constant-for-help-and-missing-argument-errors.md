---
id: d-use-one-vcs-commit-usage-constant-for-help-and-missing-argument-errors
kind: note
note_kind: decision
created: 2026-08-19T14:09:43Z
created_by: a-fixer-gcha7z
about: "[[t-01M0D2KPCZ5PEFXJS4B0J59Z5C]]"
github:
  issue: 748
  repo: mlnomadpy/dacli
---
# Use one VCS commit usage constant for help and missing-argument errors
## Chose
Use one VCS commit usage constant for help and missing-argument errors
## Rejected
Keep separate command-table and handler synopsis literals with a CLI parity override
## Because
The shared constant prevents help and usage-error drift while the invariant can validate the standard contract without a commit exception.
