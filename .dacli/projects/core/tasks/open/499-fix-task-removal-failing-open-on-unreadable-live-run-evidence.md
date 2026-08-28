---
id: t-01M0Z52NJVWN21M7HA7Y47RFS0
kind: task
created: 2026-08-26T13:45:13Z
created_by: a-adversarial-reviewer-gstekv
owner: a-root
estimate: "{optimistic: 2, probable: 3, pessimistic: 5}"
github:
  issue: 785
  repo: mlnomadpy/dacli
---
# Fix task removal failing open on unreadable live-run evidence
## Acceptance
- [ ] internal/store exposes an error-aware live-owner/live-claim scan, and task rm refuses without deleting or tombstoning when the runs directory or a recorded proc.txt is unreadable or malformed
- [ ] regressions cover both unreadable runs state and malformed proc.txt for read-write root task rm --force against a child-owned task, asserting exit 3 and the original task remains
- [ ] the readable dead-owner removal path and readable live-owner refusal remain covered and pass
- [ ] gofmt -l ., go vet ./..., golangci-lint run, and go test ./... pass; mutation making unreadable run evidence appear dead makes the new regression fail
## Log
- 2026-08-26T13:45:57Z takeover by a-root from a-adversarial-reviewer-gstekv (recovery: task takeover --force; reason: reviewer finished after filing distinct task-removal safety defect; no recovery lease remains)
