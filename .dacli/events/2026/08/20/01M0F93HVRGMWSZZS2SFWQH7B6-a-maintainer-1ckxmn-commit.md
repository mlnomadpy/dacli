---
id: 01M0F93HVRGMWSZZS2SFWQH7B6
kind: event
schema_version: 1
event_kind: commit
created: 2026-08-20T09:47:45Z
created_by: a-maintainer-1ckxmn
about: "[[t-01M0F8JAH5CNJ327M31B1821BF]]"
origin: agent
applied: true
checksum: sha256:0738786990f1da88f90d4915540baa502219117c0574a6f8468454db5dfb72a6
---
0881d1a t-01M0F8JAH5CNJ327M31B1821BF: decouple behavioral preflight from usage parsing

Persist a versioned Codex launch strategy and conservatively infer it for exact legacy exec contracts so mature workspaces retain their role references and behavioral gate. Keep custom adapters unsupported and key cached evidence by the effective strategy version.

Mutation: restoring rt.UsageFormat == "codex-jsonl" made TestLegacyCodexExecWithoutUsageFormatRunsBehavioralPreflight fail at preflight_test.go:100 with "legacy handshake = unsupported".
role: maintainer
