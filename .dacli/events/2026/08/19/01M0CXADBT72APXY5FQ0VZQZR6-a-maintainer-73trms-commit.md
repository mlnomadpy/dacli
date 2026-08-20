---
id: 01M0CXADBT72APXY5FQ0VZQZR6
kind: event
schema_version: 1
event_kind: commit
created: 2026-08-19T11:43:18Z
created_by: a-maintainer-73trms
about: "[[t-01M0AEG5F23TRH6BAR9HT38ZP1]]"
origin: agent
applied: true
checksum: sha256:b19642e5bfb46a994c4c29ba473fb9baac723626c92034aacbb8fc085a501186
---
55407f6 t-01M0AEG5F23TRH6BAR9HT38ZP1: fail closed on journal persistence

Make every recovery-ledger transition checked and validated while retaining best-effort diagnostic snapshots.

Mutation: discarding the saveState journal error fails TestSaveStatePropagatesJournalFailureButSnapshotsRemainAdvisory at journal_test.go:110 with saveState error = <nil>, want journal error.
role: maintainer
