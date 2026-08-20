---
id: f-mature-codex-regression-catches-restoration-of-the-usage-format-gate
kind: note
note_kind: finding
created: 2026-08-20T09:46:37Z
created_by: a-maintainer-1ckxmn
about: "[[t-01M0F8JAH5CNJ327M31B1821BF]]"
severity: major
---
# Mature Codex regression catches restoration of the usage-format gate
Temporarily changing internal/features/execution/behavioral_preflight.go:43 back to rt.UsageFormat == codex-jsonl made GOCACHE=/tmp/dacli-go-cache-492 go test ./internal/features/execution -run '^TestLegacyCodexExecWithoutUsageFormatRunsBehavioralPreflight$' fail at preflight_test.go:100 with 'legacy handshake = unsupported/: adapter declares no behavioral launch strategy'; restoring the strategy gate returns green.
