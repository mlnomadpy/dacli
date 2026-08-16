---
id: 01M05S10T844NTXCPK3ZYVRV2Q
kind: event
schema_version: 1
event_kind: commit
created: 2026-08-16T17:13:35Z
created_by: a-maintainer-gcwx9y
about: "[[t-01KZZVWMD0KYAPDN9QMQDK1GF3]]"
origin: agent
applied: true
checksum: sha256:16644c53b8d2e52b5e9858ed81c78fd1be600234d3c9f934ee62f97570e9b118
---
0b54699 t-01KZZVWMD0KYAPDN9QMQDK1GF3: preserve task worktree on resume

Resolve no-flag spawns from a registered same-task checkout and refuse ambiguous or mismatched worktrees with explicit recovery commands. Preserve the worktree prompt, run record, and escape checks for resumed agents.

Mutation: returning w.Root instead made TestSpawnFromTaskWorktreeRunsAndEditsOnlyThere fail: runtime-branch.txt was absent from the task worktree.
role: maintainer
