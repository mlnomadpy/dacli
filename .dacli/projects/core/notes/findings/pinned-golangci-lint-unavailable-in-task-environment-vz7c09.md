---
id: f-pinned-golangci-lint-unavailable-in-task-environment-vz7c09
kind: note
note_kind: finding
created: 2026-08-19T13:36:13Z
created_by: a-maintainer-ycbqam
about: "[[t-01M0CZAN6QKQC26961BMZMF79N]]"
severity: moderate
---
# Pinned golangci-lint unavailable in task environment
Verification on 2026-08-19: go build ./..., go vet ./..., gofmt -l ., and go test ./... passed with GOCACHE=/tmp/dacli-go-cache; golangci-lint run could not execute because the pinned binary is not installed (zsh: command not found).
