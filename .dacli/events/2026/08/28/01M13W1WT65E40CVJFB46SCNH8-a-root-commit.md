---
id: 01M13W1WT65E40CVJFB46SCNH8
kind: event
schema_version: 1
event_kind: commit
created: 2026-08-28T09:43:42Z
created_by: a-root
about: "[[t-01M13V42QDZE7CKYDFWYVB5YG5]]"
origin: agent
applied: true
checksum: sha256:b7d31ee4b687f10ef1eb9c69296f8ba0d40976651c94ff5daa8774245219c031
---
2e3d6968 t-01M13V42QDZE7CKYDFWYVB5YG5: prove supervised recovery commits

Add a public-binary regression that makes a supervised correction child commit inside a root-reclaimed task worktree under its exact claim.

Mutation evidence: omitting the supervised run worktree record fails TestSuperviseCorrectionCanCommitInRootReclaimedWorktree with exit 3: worktree owned by a-root as the correction child.
role: root
