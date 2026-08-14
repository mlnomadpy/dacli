---
id: 01KZZT7Y8M61PFKJQ09Q3M4NBB
kind: event
schema_version: 1
event_kind: commit
created: 2026-08-14T09:39:24Z
created_by: a-maintainer-ras7pq
about: "[[t-01KZZSD1K4YT88J0YYB5ZPD75R]]"
origin: agent
applied: true
checksum: sha256:582adba4b895af0e51e9c26e5a6eb30d5d6ca9fbe85f0c0663ed219c2e7d5f71
---
5c09678 t-01KZZSD1K4YT88J0YYB5ZPD75R: add audited proposal dismissal

Preserve obsolete proposals while appending one terminal disposition, enforce author/owner/root authorization, and exclude dismissed events from pending consumers and task references.

Mutation proof: disabling eventdisp indexing made TestEventsDismissAuthorizationAuditAndTaskCleanup fail with dismissed pending reference still blocked RemoveTask and named all three event files.
role: maintainer
