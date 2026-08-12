---
id: d-confirm-github-merge-state-and-persist-landing-before-best-effort-local-cleanup
kind: note
note_kind: decision
created: 2026-08-12T19:07:17Z
created_by: a-codex-maintainer-2pkfj7
about: "[[396]]"
github:
  issue: 521
  repo: mlnomadpy/dacli
---
# Confirm GitHub merge state and persist landing before best-effort local cleanup
## Chose
Confirm GitHub merge state and persist landing before best-effort local cleanup
## Rejected
Treat gh's combined merge-and-delete command error as an atomic merge failure
## Because
GitHub can commit the merge before local branch deletion fails; the remote MERGED state is authoritative, while clean-worktree removal and local branch deletion are recoverable cleanup debt.
