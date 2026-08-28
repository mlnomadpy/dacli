---
id: 01M147AHGT8EJJN2WKQR91KT8A
kind: event
schema_version: 1
event_kind: commit
created: 2026-08-28T13:00:40Z
created_by: a-maintainer-k0gy77
about: "[[t-01M146BA62817V08T9P6D6REKT]]"
origin: agent
applied: true
checksum: sha256:36e6817cb71d9596c333e87dbd67a960b9a2e762e5e8d4a246aa91088641d0bb
---
26044366 t-01M146BA62817V08T9P6D6REKT: add canonical delivery reconciliation

Derive versioned local, git, run, event, loop, scheduling, and GitHub classifications in store so doctor and future recovery consumers can share one read-only truth without slice imports.

Mutation proof: changing CLOSED PR classification to MERGED made TestReconcileDistinctFixtureClassificationsAndReadOnlyDigest fail with: missing closed-unmerged-pr.
role: maintainer
