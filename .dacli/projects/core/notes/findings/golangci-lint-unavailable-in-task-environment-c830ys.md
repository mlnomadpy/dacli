---
id: f-golangci-lint-unavailable-in-task-environment-c830ys
kind: note
note_kind: finding
created: 2026-08-19T13:36:36Z
created_by: a-maintainer-1cw4s7
about: "[[t-01M0AEYFXAB22RE9Y2SH9WZZKR]]"
severity: minor
---
# golangci-lint unavailable in task environment
The required final command reached  after gofmt, go build, serial go test ./..., and go vet passed, but zsh reported command not found. CONTRIBUTING pins v2.12.2; network/sandbox installation was not available, so lint remains unverified locally.
