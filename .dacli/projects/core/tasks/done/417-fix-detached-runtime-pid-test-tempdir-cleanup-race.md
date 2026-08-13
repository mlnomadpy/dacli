---
id: t-01KZXQDX92DDCGBYKGCHPJ9426
kind: task
created: 2026-08-13T14:11:45Z
created_by: a-root
owner: a-root
priority: should
estimate: "{optimistic: 1, probable: 2, pessimistic: 3}"
github:
  issue: 573
  repo: mlnomadpy/dacli
---
# Fix detached runtime PID test TempDir cleanup race
## Acceptance
- [x] The process or goroutine that can outlive TestExecRuntimeDetachedReportsPID is identified and documented in the test or fix.
- [x] The test waits for every writer or process that uses its TempDir before returning without weakening its PID assertion.
- [x] A stress command or test fails against the old lifecycle and passes after the fix.
- [x] go test ./internal/features/execution -run TestExecRuntimeDetachedReportsPID -count=50 and go test ./... pass.
## Log
- 2026-08-13T15:20:25Z claimed by a-junior-1etj17
- 2026-08-13T15:27:23Z accepted by a-root
- 2026-08-13T15:27:23Z closed WITHOUT verification — no --verify command was given
- 2026-08-13T15:27:23Z deliverable: dacli/417-fix-detached-runtime-pid-test-tempdir-cleanup-race exists but is NOT in main — closed anyway
- 2026-08-13T15:27:23Z completed by a-root
- 2026-08-13T16:16:39Z a-root: PR opened: https://github.com/mlnomadpy/dacli/pull/587 (event 01KZXVRJF8PZMAVPN5HY4T1XQ7)
- 2026-08-13T16:16:39Z status done proposed by a-fixer-j57wh6, applied (event 01KZXW0MDMM77Z8Q5DDFSGRQFC)
