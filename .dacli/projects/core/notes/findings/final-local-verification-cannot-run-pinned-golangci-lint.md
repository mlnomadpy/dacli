---
id: f-final-local-verification-cannot-run-pinned-golangci-lint
kind: note
note_kind: finding
created: 2026-08-19T12:50:25Z
created_by: a-maintainer-6b3z6s
about: "[[t-01M0CX031NDQ5PQ8VRX1PQNWXE]]"
severity: moderate
---
# Final local verification cannot run pinned golangci-lint
command -v golangci-lint returned unavailable in the isolated worktree. gofmt -l ., go build ./..., go vet ./..., and go test ./... passed with GOCACHE/GOMODCACHE under /tmp; CI must supply the pinned CONTRIBUTING.md version.
