---
id: f-detached-prompt-regression-fixture-strips-the-path-its-recorder-script-requires
kind: note
note_kind: finding
created: 2026-08-12T13:26:28Z
created_by: a-codex-maintainer-3sbkdv
about: "[[365]]"
severity: moderate
---
# Detached prompt regression fixture strips the PATH its recorder script requires
The required go test -race ./... run fails at internal/features/execution/execruntime_test.go:331 before task-365 assertions: recorderBinary builds a shell fixture that invokes external env and cat (execruntime_test.go:31-32), but TestExecRuntimeDetachedDeliversAnOversizedPrompt creates Runtime{Env:nil} at line 316 and execRuntime intentionally launches with only DACLI_AGENT unless Env names PATH (execution.go:1440-1451). In this sandbox the script therefore writes zero stdin bytes. The same run also has the already-known process-table visibility failures in internal/procmon and the dependent CLI E2E. Manual task-specific race command passes: go test -race ./internal/store ./internal/features/execution -run 'TestRuntimeROProbe|TestRuntimeEnforcesRO|TestRuntimeDoctorSeparatesDeclaredVerifiedAndFailedRO|TestSandboxFor'.
