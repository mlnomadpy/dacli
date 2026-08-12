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
- [x] env GOCACHE=/private/tmp/dacli-go-cache go test ./internal/cli -run TestE2EFixtureRepoGoesFromEmptyToShipped -count=1 passes in the restricted agent sandbox, or skips only after an explicit reproducible unsupported-platform premise check
- [x] When the fixture worker exits non-zero, internal/cli/e2e_fixture_test.go includes the child transcript or equivalent root-cause stderr in the failure output before t.TempDir cleanup
- [x] A regression test or controlled failure proves the diagnostic assertion goes red when worker stderr is discarded and distinguishes external sandbox startup refusal from a dacli coordination failure
- [x] env GOCACHE=/private/tmp/dacli-go-cache go test ./... passes in the restricted agent sandbox
## Log
- 2026-08-12T18:23:28Z adopted by a-root (owner a-codex-loop-auditor-f6h2e4 orphaned)
- 2026-08-12T18:24:00Z claimed by a-codex-maintainer-f85g9w
- 2026-08-12T18:30:33Z claimed by a-root (event 01KZVKD970AG7MM74NA7VNY7D3)
- 2026-08-12T18:56:43Z a-root: PR opened: https://github.com/mlnomadpy/dacli/pull/510 (event 01KZVKXYPK7SY8PC7D0EQ317HY)
- 2026-08-12T19:00:46Z accepted by a-root (applied 1 proposal(s))
- 2026-08-12T19:00:46Z verified by `GOCACHE=/private/tmp/dacli-gocache GOTMPDIR=/private/tmp go test ./internal/cli -run TestE2EFixtureRepoGoesFromEmptyToShipped -count=1` (exit 0) in branch main at 6ea0f9e — proves that tree builds, not that the work is in trunk
- 2026-08-12T19:00:46Z deliverable: dacli/391-make-the-self-hosting-e2e-fixture-diagnose-and-handle-restricted-worker-startup exists but is NOT in main — closed anyway
- 2026-08-12T19:00:46Z completed by a-root
- 2026-08-12T19:44:07Z accepted by a-root
- 2026-08-12T19:44:07Z closed WITHOUT verification — no --verify command was given
- 2026-08-12T19:44:07Z deliverable: dacli/391-make-the-self-hosting-e2e-fixture-diagnose-and-handle-restricted-worker-startup is merged into main
- 2026-08-12T19:44:07Z completed by a-root
