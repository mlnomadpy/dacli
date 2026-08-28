---
id: f-pinned-golangci-lint-unavailable-in-task-sandbox-vwynvy
kind: note
note_kind: finding
created: 2026-08-28T13:52:30Z
created_by: a-maintainer-wg7dnx
about: "[[t-01M12QX9HEPKAAS1033W6HS45D]]"
severity: minor
---
# Pinned golangci-lint unavailable in task sandbox
CONTRIBUTING.md pins golangci-lint v2.12.2. The binary is absent (env: golangci-lint: No such file or directory), and go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.12.2 run cannot install it because proxy.golang.org DNS is unavailable. go build ./..., env -u DACLI_AGENT go test ./..., go vet ./..., gofmt -l ., git diff --check, and focused mutation/regression checks completed successfully.
