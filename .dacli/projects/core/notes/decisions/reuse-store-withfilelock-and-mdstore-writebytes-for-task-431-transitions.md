---
id: d-reuse-store-withfilelock-and-mdstore-writebytes-for-task-431-transitions
kind: note
note_kind: decision
created: 2026-08-13T20:32:28Z
created_by: a-codex-maintainer-ttvrdm
about: "[[431]]"
github:
  issue: 622
  repo: mlnomadpy/dacli
---
# Reuse store.WithFileLock and mdstore.WriteBytes for task 431 transitions
## Chose
Reuse store.WithFileLock and mdstore.WriteBytes for task 431 transitions
## Rejected
Retain feature-local PID locks and temp-file rename writers
## Because
The shared primitives already provide token-owned stale-lock recovery and directory-fsynced durable replacement, eliminating weaker duplicate implementations while preserving feature-local receipt schemas.
