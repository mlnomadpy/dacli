---
id: f-128-manually-verified-goos-goarch-build-matrix-goreleaser-snapshot-run-blocked
kind: note
note_kind: finding
created: 2026-07-23T19:39:24Z
created_by: a-z2xq5q3axy
about: [[128]]
severity: minor
---
# 128: manually verified GOOS/GOARCH build matrix; goreleaser snapshot run blocked by sandbox
Verified 'go build ./cmd/dacli' succeeds for windows/amd64, windows/arm64, darwin/amd64, darwin/arm64, linux/amd64, linux/arm64 (the exact matrix .goreleaser.yaml already declares at builds[0].goos/goarch). go build ./..., go vet ./..., GOOS=windows go vet ./..., and go test -exec 'env -u DACLI_AGENT' ./internal/... all green on host (darwin). Could NOT run the literal 'goreleaser release --snapshot --clean' acceptance check: goreleaser is not preinstalled, and both 'go install github.com/goreleaser/goreleaser/v2@latest' (pulled a huge unrelated dependency tree, did not finish) and directly invoking the resulting/any goreleaser binary were blocked by this headless sandbox's command-approval gate (which has no one to approve it). Manual per-target 'go build ./cmd/dacli' is the same build goreleaser's builds: stage runs per (goos,goarch) pair, so this is strong but not literal evidence -- owner should run the snapshot release once to confirm archive packaging (esp. windows .zip via format_overrides) also succeeds.
