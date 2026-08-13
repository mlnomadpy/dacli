---
id: d-reuse-dacli-durable-writes-and-robust-store-locks-for-transition-receipts
kind: note
note_kind: decision
created: 2026-08-13T20:25:07Z
created_by: a-root
about: "[[431]]"
github:
  issue: 621
  repo: mlnomadpy/dacli
---
# Reuse dacli durable writes and robust store locks for transition receipts
## Chose
Reuse dacli durable writes and robust store locks for transition receipts
## Rejected
Duplicate PID-only syscall signal locks and temp-file rename writers inside queue and stage slices
## Because
mdstore.WriteBytes already fsyncs the temporary file and containing directory, while store acquireFileLock already uses token ownership, host-aware stale detection, and atomic O_EXCL creation; export a narrowly named store lock wrapper and use these shared lower-layer primitives so task 431 is crash-durable and cross-platform without parallel lock implementations
