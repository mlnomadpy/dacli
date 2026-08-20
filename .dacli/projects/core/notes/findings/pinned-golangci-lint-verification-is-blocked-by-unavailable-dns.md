---
id: f-pinned-golangci-lint-verification-is-blocked-by-unavailable-dns
kind: note
note_kind: finding
created: 2026-08-19T14:11:42Z
created_by: a-fixer-gcha7z
about: "[[t-01M0D2KPCZ5PEFXJS4B0J59Z5C]]"
severity: moderate
---
# Pinned golangci-lint verification is blocked by unavailable DNS
golangci-lint is absent from PATH. Installing the pinned v2.12.2 into /tmp failed because proxy.golang.org could not resolve; gofmt, go vet, go build, focused tests, and go test ./... passed.
