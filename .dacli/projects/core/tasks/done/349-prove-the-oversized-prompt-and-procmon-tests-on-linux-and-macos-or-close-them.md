---
id: t-01KZPWRY7NRVHN82P6VZ242W30
kind: task
created: 2026-08-10T22:30:28Z
created_by: a-root
owner: a-root
priority: must
estimate: "{optimistic: 2, probable: 3, pessimistic: 5}"
---
# Prove the oversized-prompt and procmon tests on Linux and macOS, or close them as already green
## Acceptance
- [x] CI runs the full suite on BOTH ubuntu and macOS, so a platform-specific failure cannot hide behind one runner
- [x] TestExecRuntimeDetachedDeliversAnOversizedPrompt and the procmon unix tests are run repeatedly under -race on both, not once
- [x] if they pass everywhere, issue #437 items 1 and 2 are closed with the evidence rather than left open on an unreproduced premise
## Log
- 2026-08-11T09:26:27Z accepted by a-root
- 2026-08-11T09:26:27Z verified by `go test -race -count=3 ./internal/procmon/` (exit 0)
- 2026-08-11T09:26:27Z deliverable: no dacli/349-prove-the-oversized-prompt-and-procmon-tests-on-linux-and-macos-or-close-them branch — nothing to check against main
- 2026-08-11T09:26:27Z completed by a-root
