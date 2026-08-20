---
id: 01M0D5ZB7SF9KSKQ3TMX155XXE
kind: event
schema_version: 1
event_kind: commit
created: 2026-08-19T14:14:32Z
created_by: a-fixer-gcha7z
about: "[[t-01M0D2KPCZ5PEFXJS4B0J59Z5C]]"
origin: agent
applied: true
checksum: sha256:7cf854dd2843d5d107f6ffaa0d412610c6de5836dbc3ef5d5a5c3ac80b219af2
---
53767a9 t-01M0D2KPCZ5PEFXJS4B0J59Z5C: restore canonical commit help contract

Share the VCS commit synopsis between command help and the missing-argument error so the CLI parity invariant needs no commit exception. Mutation: restoring the truncated table literal fails TestCommitUsageMatchesCommandTable at commit_test.go:35.
role: fixer
