---
id: 01KZX67NX3WQ4VBP4YABPG0QAY
kind: event
event_kind: commit
created: 2026-08-13T09:11:15Z
created_by: a-root
about: "[[t-01KZX5XZ9VHSM4X1ST5JA2M9JA]]"
origin: agent
applied: true
---
522a841 fix merged PR lookup after head deletion

Reproduced: task 401's recorded merged PR #544 was reported orphaned once GitHub deleted its head branch.

Full gates: gofmt, go vet ./..., golangci-lint run, go test ./...
role: root
