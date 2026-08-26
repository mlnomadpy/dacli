---
id: f-pinned-golangci-lint-is-unavailable-in-this-environment
kind: note
note_kind: finding
created: 2026-08-26T14:37:21Z
created_by: a-fixer-1x0gq5
about: "[[t-01M0F8DMCN93FCDE59FSEDTJB3]]"
severity: moderate
---
# Pinned golangci-lint is unavailable in this environment
gofmt -l . and GOCACHE=/private/tmp/dacli-go-cache go vet ./... completed cleanly, but golangci-lint run exited 127: command not found. The required pinned v2.12.2 binary is not installed and network installation is unavailable in this sandbox.
