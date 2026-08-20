---
id: d-replace-patch-equivalent-stale-task-refs-only-under-an-exact-observed-oid-lease
kind: note
note_kind: decision
created: 2026-08-19T14:12:38Z
created_by: a-maintainer-4243q0
about: "[[t-01M0D2KPGZZMYYSVSHNB8NS2T9]]"
github:
  issue: 747
  repo: mlnomadpy/dacli
---
# Replace patch-equivalent stale task refs only under an exact observed-OID lease
## Chose
Replace patch-equivalent stale task refs only under an exact observed-OID lease
## Rejected
Continue rebasing task branches onto their remote tip, or force-update every divergent task branch
## Because
Rebase recreates issue #726; an unconditional force can discard unique remote work. git cherry proves remote-only patches are on fetched trunk, while merge-base proves the local branch starts at current trunk; all other divergence refuses without moving HEAD.
