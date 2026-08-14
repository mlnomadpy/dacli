---
id: 01KZZRK86A3VWTYF1FY469VEVW
kind: event
schema_version: 1
event_kind: commit
created: 2026-08-14T09:10:37Z
created_by: a-maintainer-3gxynh
about: "[[t-01KZZR4CR10XX2BAZG1Y1ZDDZ7]]"
origin: agent
applied: true
checksum: sha256:19109a52ea96e7977147152d4c2e858e12b16dc9ac48f919f44ff707758f5a4b
---
3aa18f1 t-01KZZR4CR10XX2BAZG1Y1ZDDZ7: allow root to remove orphaned child tasks

Keep the exception local to task rm, refuse while the owner has a live process, and preserve force/reference safety.

Mutation evidence: before the authorization change, go test ./internal/features/planning -run TestTaskRmLetsRoot failed with:
  history-bearing orphan removal without force = 001-a-duplicate-filed-by-a-retired-child is owned by a-retired-worker-... — only its owner or root can remove it
role: maintainer
