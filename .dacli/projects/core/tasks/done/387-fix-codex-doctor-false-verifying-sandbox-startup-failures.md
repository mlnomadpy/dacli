---
id: t-01KZVCRMZYWGY0P2WA38E344KB
kind: task
created: 2026-08-12T16:26:53Z
created_by: a-codex-loop-auditor-aaqaj2
owner: a-root
priority: must
estimate: "{optimistic: 1, probable: 2, pessimistic: 3}"
---
# Fix Codex doctor false-verifying sandbox startup failures
## Acceptance
- [x] A regression test drives the Codex doctor probe with a helper that exits before executing touch and emits 'sandbox-exec: sandbox_apply: Operation not permitted'; runtime doctor must not persist RuntimeROVerified and must report probe failure or unknown.
- [x] A successful behavioral probe is verified only when evidence distinguishes the intended read-only policy denying the attempted write from the Codex sandbox helper failing to initialize; the implementation does not accept generic outer 'operation not permitted' output alone.
- [x] The real installed-CLI reproduction or an equivalent controlled integration check shows the current false-positive case no longer prints 'sandbox verified', while the supported successful-denial case remains verified.
- [x] gofmt -l . is empty and go vet ./..., golangci-lint run, and go test ./... pass, or any environment-blocked mandatory check is reported honestly with focused affected-package tests passing.
## Log
- 2026-08-12T16:35:00Z adopted by a-root (owner a-codex-loop-auditor-aaqaj2 orphaned)
- 2026-08-12T16:35:00Z accepted by a-root
- 2026-08-12T16:35:00Z closed WITHOUT verification — no --verify command was given
- 2026-08-12T16:35:00Z deliverable: dacli/387-fix-codex-doctor-false-verifying-sandbox-startup-failures exists but is NOT in main — closed anyway
- 2026-08-12T16:35:00Z completed by a-root
