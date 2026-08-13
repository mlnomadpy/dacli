---
id: d-bind-isolated-worktree-commits-to-the-recorded-spawned-child-before-staging
kind: note
note_kind: decision
created: 2026-08-13T19:06:19Z
created_by: a-codex-maintainer-mjejj8
about: "[[427]]"
github:
  issue: 599
  repo: mlnomadpy/dacli
---
# Bind isolated worktree commits to the recorded spawned child before staging
## Chose
Bind isolated worktree commits to the recorded spawned child before staging
## Rejected
Rely only on commit message validation or current process liveness
## Because
Message validation blocks the observed -m spelling but not a valid-looking lost-token owner invocation; worktree.txt plus proc child remains stable through guardian exit and protects attribution without racing lifecycle completion.
