---
id: t-01KY7ENG3ARAJV6FMAACMQ0M4P
kind: task
created: 2026-07-23T12:19:37Z
created_by: a-m146x20e8d
owner: a-m146x20e8d
priority: should
---
# github pull: skip human-closed issues so inbound sync doesn't resurrect settled work
## So that
a maintainer who closes an issue as wontfix/duplicate/resolved does not have it re-adopted as a fresh open task on the next pull/sync
## Acceptance
- [ ] shouldImport (internal/features/ghmirror/ghmirror.go) returns false for an issue whose State is 'closed' unless it is already mapped to a local task, so github pull skips human-closed issues instead of adopting them as new open tasks
- [ ] A new unit test in ghmirror_test.go (alongside TestShouldImportSkipLogic) asserts a closed human-authored, unmapped issue is NOT imported, while an open one still is; go test ./internal/... stays green
## Log
