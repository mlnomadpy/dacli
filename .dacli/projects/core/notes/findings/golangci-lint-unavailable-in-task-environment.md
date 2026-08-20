---
id: f-golangci-lint-unavailable-in-task-environment
kind: note
note_kind: finding
created: 2026-08-19T12:12:37Z
created_by: a-maintainer-6w2h0v
about: "[[t-01M0AF65RDNBEX2SEF9JC5RTMZ]]"
severity: minor
---
# golangci-lint unavailable in task environment
The required pinned lint command could not be run because golangci-lint is not installed (zsh: command not found). go build ./..., go test ./..., go vet ./..., and gofmt -l . all passed.
