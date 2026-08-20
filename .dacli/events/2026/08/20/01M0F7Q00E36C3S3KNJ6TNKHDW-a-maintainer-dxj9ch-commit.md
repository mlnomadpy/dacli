---
id: 01M0F7Q00E36C3S3KNJ6TNKHDW
kind: event
schema_version: 1
event_kind: commit
created: 2026-08-20T09:23:25Z
created_by: a-maintainer-dxj9ch
about: "[[t-01M0F3795JGCAG6ZS3XVAGNS2J]]"
origin: agent
applied: true
checksum: sha256:07a14dc66c3925fe83bd067ad61175e7a544c980b3738d89658c22b5368a5d27
---
3512622 t-01M0F3795JGCAG6ZS3XVAGNS2J: preflight exact runtime launch compatibility

Codex sandbox verification did not prove app-server startup in the effective sandbox. Add an adapter-owned bounded handshake and exact fresh cache before spawn records are minted.

Mutation: removing requireLaunchCompatibility made TestCodexROBehavioralPreflightRefusesBeforeSpawnRecords fail at preflight_test.go:56 with exit 1 worktree add, want the preflight exit-3 refusal.
role: maintainer
