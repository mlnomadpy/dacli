---
id: f-internal-features-catalog-tests-leak-dacli-agent-into-in-process-commands
kind: note
note_kind: finding
created: 2026-08-04T00:35:59Z
created_by: a-fixer-yanvqd
about: "[[200]]"
severity: minor
---
# internal/features/catalog tests leak DACLI_AGENT into in-process commands
go test ./internal/features/catalog/... fails TestCatalogRefusesRatherThanWritingAnEmptyRoster with 'agent token not recognized in this workspace' when run from inside a dacli agent session, because rosterwipe_test.go's in-process command calls read the ambient DACLI_AGENT env var. internal/cli already fixed this class via TestMain clearing DACLI_AGENT (see f-021-verified-...); internal/features/catalog has no equivalent TestMain. Not caused by this task's change (docs-only, .dacli/roles/*.md); pre-existing test-isolation gap, out of scope for 200.
