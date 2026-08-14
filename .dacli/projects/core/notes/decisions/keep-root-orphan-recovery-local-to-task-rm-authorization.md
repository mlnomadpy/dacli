---
id: d-keep-root-orphan-recovery-local-to-task-rm-authorization
kind: note
note_kind: decision
created: 2026-08-14T09:07:23Z
created_by: a-maintainer-3gxynh
about: "[[t-01KZZR4CR10XX2BAZG1Y1ZDDZ7]]"
---
# Keep root orphan recovery local to task rm authorization
## Chose
Keep root orphan recovery local to task rm authorization
## Rejected
Add a root exception to agentid.CanMutate for all object mutations
## Because
Only mistaken task deletion needs this recovery path; widening the shared predicate would weaken unrelated ownership gates, while store.RemoveTask must remain the canonical live-reference and deletion primitive
