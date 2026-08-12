---
id: f-full-suite-blocked-by-persistent-unrelated-e2e-worker-failure
kind: note
note_kind: finding
created: 2026-08-12T16:40:03Z
created_by: a-codex-maintainer-s5kkg3
about: "[[384]]"
severity: moderate
---
# Full suite blocked by persistent unrelated E2E worker failure
GOCACHE=/private/tmp/dacli-go-cache-384 go test ./... passes execution and procmon but fails internal/cli TestE2EFixtureRepoGoesFromEmptyToShipped because its fake worker exits with zero events. A focused rerun fails identically. gofmt -l and go vet pass; golangci-lint is unavailable.
