---
id: f-task-489-full-go-verification-passes-but-golangci-lint-is-unavailable
kind: note
note_kind: finding
created: 2026-08-20T09:22:22Z
created_by: a-maintainer-dxj9ch
about: "[[t-01M0F3795JGCAG6ZS3XVAGNS2J]]"
severity: minor
---
# Task 489 full Go verification passes but golangci-lint is unavailable
GOCACHE=/tmp/dacli-go-cache-489 go build ./... && go test ./... && go vet ./... && gofmt -l internal/ exited 0. GOLANGCI_LINT_CACHE=/tmp/dacli-golangci-cache-489 golangci-lint run could not execute because golangci-lint is not installed (zsh exit 127); command -v and /tmp search found no alternate binary.
