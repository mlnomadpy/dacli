---
id: t-01KZVEBZ5ATTV0C8C3B8ZSEDZ5
kind: task
created: 2026-08-12T16:54:55Z
created_by: a-codex-loop-auditor-f6h2e4
owner: a-root
priority: must
estimate: "{optimistic: 1, probable: 2, pessimistic: 3}"
github:
  issue: 488
  repo: mlnomadpy/dacli
---
# Make the self-hosting E2E fixture diagnose and handle restricted worker startup
## Acceptance
- [ ] env GOCACHE=/private/tmp/dacli-go-cache go test ./internal/cli -run TestE2EFixtureRepoGoesFromEmptyToShipped -count=1 passes in the restricted agent sandbox, or skips only after an explicit reproducible unsupported-platform premise check
- [ ] When the fixture worker exits non-zero, internal/cli/e2e_fixture_test.go includes the child transcript or equivalent root-cause stderr in the failure output before t.TempDir cleanup
- [ ] A regression test or controlled failure proves the diagnostic assertion goes red when worker stderr is discarded and distinguishes external sandbox startup refusal from a dacli coordination failure
- [ ] env GOCACHE=/private/tmp/dacli-go-cache go test ./... passes in the restricted agent sandbox
## Log
- 2026-08-12T18:23:28Z adopted by a-root (owner a-codex-loop-auditor-f6h2e4 orphaned)
- 2026-08-12T18:24:00Z claimed by a-codex-maintainer-f85g9w
- 2026-08-12T18:30:33Z claimed by a-root (event 01KZVKD970AG7MM74NA7VNY7D3)
