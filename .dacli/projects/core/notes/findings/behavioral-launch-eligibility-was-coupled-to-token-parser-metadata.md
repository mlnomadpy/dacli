---
id: f-behavioral-launch-eligibility-was-coupled-to-token-parser-metadata
kind: note
note_kind: finding
created: 2026-08-20T09:43:21Z
created_by: a-maintainer-1ckxmn
about: "[[t-01M0F8JAH5CNJ327M31B1821BF]]"
severity: major
---
# Behavioral launch eligibility was coupled to token parser metadata
internal/features/execution/behavioral_preflight.go:43 enabled the Codex startup handshake only when UsageFormat equaled codex-jsonl; internal/store/runtimefiles.go's launch fingerprint likewise keyed usage parsing instead of an execution capability, so a mature adapter without usage_format bypassed the gate.
