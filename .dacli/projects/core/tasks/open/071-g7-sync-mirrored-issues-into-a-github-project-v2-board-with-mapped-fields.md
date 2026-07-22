---
id: t-01KY5JKS7ASVC1QM7HJNVYGV2E
kind: task
created: 2026-07-22T18:50:06Z
created_by: a-root
owner: a-root
priority: should
estimate: {optimistic: 5, probable: 8, pessimistic: 13}
---
# G7: sync mirrored issues into a GitHub Project (v2) board with mapped fields
## Acceptance
- [ ] dacli can create/link a GitHub Project v2 for the repo and add every mirrored issue to it, operator-triggered and disclosure-gated like the other projections
- [ ] dacli fields map to Project fields: Status (from task folder / finding), Severity (from finding severity), Area (from area: label); idempotent — re-run does not duplicate items
- [ ] uses gh project (create/item-add/field-*); no live gh in tests (unit-test the mapping); committed by an agent; build + test green
## Log
