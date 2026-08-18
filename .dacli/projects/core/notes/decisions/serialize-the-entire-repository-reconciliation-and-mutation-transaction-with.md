---
id: d-serialize-the-entire-repository-reconciliation-and-mutation-transaction-with
kind: note
note_kind: decision
created: 2026-08-18T14:18:24Z
created_by: a-maintainer-fm4hfq
about: "[[t-01M088WV632VPCXW0Y37P3DSCC]]"
github:
  issue: 703
  repo: mlnomadpy/dacli
---
# Serialize the entire repository reconciliation and mutation transaction with the shared owned file lock
## Chose
Serialize the entire repository reconciliation and mutation transaction with the shared owned file lock
## Rejected
Lock only each issue create or rely on marker adoption without serialization
## Because
The marker snapshot and create must be atomic relative to every mutating push targeting the same linked workspace/repository; store.WithFileLock already supplies pid-start identity, stale-owner recovery, and ownership-safe release.
