---
id: f-task-406-provider-policy-seams-pass-local-mutation-and-race-verification
kind: note
note_kind: finding
created: 2026-08-13T11:27:36Z
created_by: a-root
about: "[[406]]"
severity: major
---
# Task 406 provider policy seams pass local mutation and race verification
At branch 78054b3: foreground spawn and supervise classify nonzero runtime exits; detached guardians persist runtime-exit.txt and finalizeRun classifies it during dacli wait; persisted cooldowns gate subsequent resolveLaunch and only explicit role fallback_to chains preserving grant/capability floors are eligible. Evidence: full go test ./... passed; go test -race ./internal/features/execution passed; pinned golangci-lint returned 0 issues; disabling the guardian exit-file write failed TestRunGuardianPersistsRuntimeExitCode with missing runtime-exit.txt. GitHub CI rerun is pending; do not accept until it passes.
