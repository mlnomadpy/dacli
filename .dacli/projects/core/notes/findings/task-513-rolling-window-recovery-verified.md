---
id: f-task-513-rolling-window-recovery-verified
kind: note
note_kind: finding
created: 2026-08-27T11:41:59Z
created_by: a-root
about: "[[513]]"
severity: major
origin: internal/features/orchestration/orchestration.go:335
---
# Task 513 rolling-window recovery verified
Reproduced on the persisted production shape: window_start zero with window_spent 3020977 and a newly supplied 240000 ceiling returned sleep-window for 24h. Regression failed before the fix. Resolution resets spend only when the durable cycle journal proves no prior ceiling; active and previously capped windows preserve spend. Dry-run previews the same decision without persisting it. PASS: gofmt -l ., go vet ./..., golangci-lint run (0 issues), go test ./....
