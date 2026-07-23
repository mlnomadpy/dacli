---
id: f-113-claude-code-preset-now-defaults-usage-format-stream-json-doctor-warns-when
kind: note
note_kind: finding
created: 2026-07-23T22:12:34Z
created_by: a-gczzw3m5kd
about: [[113]]
severity: moderate
---
# 113: claude-code preset now defaults usage_format: stream-json; doctor warns when a claude binary has none
internal/features/execution/execution.go:56-68 -- the claude-code preset now sets UsageFormat: "stream-json" so a fresh 'dacli runtime add <name> --preset claude-code' opts into stream-json capture without any extra flag; this feeds both agents --tail's isTextRuntime check and calibration's usage.txt harvest (execRuntime, execution.go:880-1016) for free on a new workspace. internal/features/execution/execution.go:177-183 -- cmdRuntimeDoctor now prints a second line ('<name> ⚠ no usage_format: --tail and calibration will be blind — enable stream-json') for any configured runtime whose Binary == "claude" and UsageFormat is empty, so an override-away-from-default or a hand-authored claude adapter is caught instead of staying silently blind. generic-exec is deliberately left with an empty UsageFormat (see decision note) since it has no fixed binary to assume a streaming shape for. docs/RUNTIMES.md updated in §22/§23 to document the new default and the doctor warning. Tests added: internal/cli/runtime_test.go TestRuntimeAddClaudeCodePresetDefaultsUsageFormatStreamJSON, TestRuntimeAddGenericExecPresetLeavesUsageFormatEmpty, TestRuntimeDoctorWarnsOnClaudeBinaryWithoutUsageFormat. gofmt -l . clean; go build ./... clean; go test ./... all green (run with DACLI_AGENT stripped per the known cli test-isolation gap).
