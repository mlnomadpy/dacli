---
id: 01M05SYP1Z7082TB1TDQSX0K9D
kind: event
schema_version: 1
event_kind: commit
created: 2026-08-16T17:29:47Z
created_by: a-maintainer-psevtg
about: "[[t-01KZYW7M979TQNHD2VTA1Q9WAT]]"
origin: agent
applied: true
checksum: sha256:234b9f5c78518aa7273d37d5525095ec8ab38a6bb2764d18621366474968ad3a
---
3efa1b9 t-01KZYW7M979TQNHD2VTA1Q9WAT: make pending accept recovery state-aware

Clear merged journal entries when the canonical task is already fully accepted, and persist an actionable verify-required state for command criteria so exit-3 acceptance is not retried.

Mutation: forcing acceptanceComplete to false made TestReconcilePendingAcceptsClearsAlreadyAcceptedMergedTask fail at driver_test.go:775: stale accepted entry survived or was counted again.
role: maintainer
