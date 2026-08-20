---
id: 01M0D604NRMMJCB8XN0GFGNZQ8
kind: event
schema_version: 1
event_kind: commit
created: 2026-08-19T14:14:59Z
created_by: a-maintainer-4243q0
about: "[[t-01M0D2KPGZZMYYSVSHNB8NS2T9]]"
origin: agent
applied: true
checksum: sha256:060b867165ba73d8e54d565aa6f1edc9e26ae487c346e8d1a8d08165307a3daf
---
9e11830 t-01M0D2KPGZZMYYSVSHNB8NS2T9: protect rebased task pushes with exact leases

Remote task history whose patches already landed on current trunk is replaced only under the fetched remote OID lease. Ambiguous divergence refuses without moving HEAD and names the exact recovery command.

Mutation: disabling the task-branch lease guard made TestPushSyncLeaseReplacesObsoleteRebasedTaskHistory fail: output must distinguish the lease-protected operation (the obsolete synchronized-push-after-rebase path ran).
role: maintainer
