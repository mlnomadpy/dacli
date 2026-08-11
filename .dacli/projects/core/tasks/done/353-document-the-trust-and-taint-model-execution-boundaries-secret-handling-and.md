---
id: t-01KZPWSHZCKTFF5GQG5E3CVC0E
kind: task
created: 2026-08-10T22:30:48Z
created_by: a-root
owner: a-root
priority: must
estimate: "{optimistic: 3, probable: 5, pessimistic: 8}"
---
# Document the trust and taint model, execution boundaries, secret handling, and rollback behaviour for every mutating surface
## Acceptance
- [x] every command declaring Mutates is covered: what it changes, what gates it, and how to undo it
- [x] the taint model and the untrusted-content boundary are stated in one place rather than inferred from code comments
- [x] secret handling says what dacli reads, what it never writes to a record, and where an agent token can and cannot appear
- [x] a test asserts the doc lists every Mutates command, so the doc cannot drift as commands are added
## Log
- 2026-08-11T10:00:19Z claimed by a-fixer-kf182p
- 2026-08-11T10:24:45Z accepted by a-root
- 2026-08-11T10:24:45Z closed WITHOUT verification — no --verify command was given
- 2026-08-11T10:24:45Z deliverable: dacli/353-document-the-trust-and-taint-model-execution-boundaries-secret-handling-and exists but is NOT in sprint/14 — closed anyway
- 2026-08-11T10:24:45Z completed by a-root
