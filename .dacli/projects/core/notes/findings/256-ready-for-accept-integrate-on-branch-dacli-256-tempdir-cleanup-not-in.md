---
id: f-256-ready-for-accept-integrate-on-branch-dacli-256-tempdir-cleanup-not-in
kind: note
note_kind: finding
created: 2026-08-04T11:54:33Z
created_by: a-maintainer-g2363w
about: "[[256]]"
severity: minor
---
# 256 ready for accept+integrate on branch dacli/256-...tempdir-cleanup-not-in (commit 017521c)
Both acceptance criteria met on branch dacli/256-testshiprecordmessagereportsactualmerges-fails-in-ci-at-tempdir-cleanup-not-in, commit 017521c (single file: internal/features/ship/ship_test.go, +33). (1) shipEnv now disables git background auto-gc/maintenance so no detached git subprocess survives to race t.TempDir cleanup; (2) the cause is named in a comment (ship_test.go:42-52), not papered over with a retry. Proof run: go build ./... OK; go vet ./... OK; gofmt -l internal/ clean; go test ./internal/features/ship/ green incl. new TestShipEnvDisablesGitAutoMaintenance. NOTE: go test ./... shows one UNRELATED pre-existing failure in internal/features/catalog (TestCatalogRefusesRatherThanWritingAnEmptyRoster: 'agent token not recognized') -- the known DACLI_AGENT env-leak because this suite runs inside a dacli agent session; catalog has no TestMain clearing DACLI_AGENT (cli does). It passes in CI where DACLI_AGENT is unset and is not caused by this change. Owner: dacli accept 256 then integrate/merge --task 256.
