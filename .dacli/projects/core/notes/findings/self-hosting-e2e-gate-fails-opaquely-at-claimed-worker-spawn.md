---
id: f-self-hosting-e2e-gate-fails-opaquely-at-claimed-worker-spawn
kind: note
note_kind: finding
created: 2026-08-12T16:54:55Z
created_by: a-codex-loop-auditor-f6h2e4
about: "[[390]]"
severity: major
---
# Self-hosting E2E gate fails opaquely at claimed worker spawn
Reproduced twice with env GOCACHE=/private/tmp/dacli-go-cache go test ./internal/cli -run TestE2EFixtureRepoGoesFromEmptyToShipped -count=1 -v: internal/cli/e2e_fixture_test.go:93 reports child exit 1 with zero events and acceptance 0/2. The reported transcript is inside t.TempDir and is removed on failure, so the child cause cannot be inspected after the test. Adjacent TestShipDoesNotStampAFalseUnlandedRecord and TestWorkerProposedDoneIsLandedByTheToolNotByTheTest pass in the same sandbox. Full go test ./... fails only internal/cli at this fixture.
