---
id: f-task-372-implementation-committed-on-its-isolated-branch
kind: note
note_kind: finding
created: 2026-08-12T14:05:48Z
created_by: a-codex-maintainer-nhx5wh
about: "[[372]]"
severity: major
---
# Task 372 implementation committed on its isolated branch
Commit 10ab5b5 on dacli/372-enforce-recorded-timeouts-for-detached-and-interrupted-runtime-trees persists timeout_s in proc.txt, launches a detached authenticated watchdog, identity-checks every recorded cleanup, preserves timed-out outcomes against wait reconciliation, and makes foreground SIGINT/SIGTERM cancel the whole process group. Focused mutation red: TestFinalizePreservesRecordedTimeout overwrote timed-out with no visible result when the guard was disabled. Full go test and go test -race could not pass in this sandbox: process-table ps probes are denied (existing procmon tests and the new real-tree fixture cannot observe PID starts), and pre-existing detached prompt/e2e fixtures fail; golangci-lint is not installed. go vet passed with a sandbox-local GOCACHE; focused executable tests passed.
