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
- [ ] The process or goroutine that can outlive TestExecRuntimeDetachedReportsPID is identified and documented in the test or fix.
- [ ] The test waits for every writer or process that uses its TempDir before returning without weakening its PID assertion.
- [ ] A stress command or test fails against the old lifecycle and passes after the fix.
- [ ] go test ./internal/features/execution -run TestExecRuntimeDetachedReportsPID -count=50 and go test ./... pass.
## Log
- 2026-08-13T15:20:25Z claimed by a-junior-1etj17
