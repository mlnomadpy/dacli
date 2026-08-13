---
id: f-local-patched-toolchain-scan-blocked-by-sandbox-dns
kind: note
note_kind: finding
created: 2026-08-13T22:46:46Z
created_by: a-fixer-8skqtd
about: "[[438]]"
severity: moderate
---
# Local patched-toolchain scan blocked by sandbox DNS
GOPATH=/private/tmp/dacli-438-gopath GOTOOLCHAIN=go1.25.13 go version attempted the official toolchain download but failed resolving proxy.golang.org; govulncheck and golangci-lint are not installed locally. Workflow contract tests, gofmt, go vet ./..., and go test ./... pass, but the patched govulncheck invocation remains for CI to execute.
