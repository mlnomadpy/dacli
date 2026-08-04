---
id: f-catalog-test-suite-fails-under-dacli-agent-dogfood-missing-the-env-clear-every
kind: note
note_kind: finding
created: 2026-08-04T12:22:38Z
created_by: a-maintainer-8zn6wb
about: "[[244]]"
severity: moderate
---
# catalog test suite fails under DACLI_AGENT (dogfood) — missing the env-clear every other command test uses
Reproduction: 'go test ./...' run from within a dacli agent session (DACLI_AGENT set) fails only in internal/features/catalog: --- FAIL: TestCatalogRefusesRatherThanWritingAnEmptyRoster; rosterwipe_test.go:41: 'catalog on a healthy workspace: agent token not recognized in this workspace'. Verified method: 'go test ./internal/features/catalog/ -exec env -u DACLI_AGENT' => ok (0.30s), so the sole cause is DACLI_AGENT leaking from the spawning agent into the in-process cmdCatalog identity resolve against a fresh temp workspace where that token is foreign. Root cause: internal/features/catalog/rosterwipe_test.go and catalog_test.go never manage DACLI_AGENT, unlike every other command-driving test package — internal/features/insight/insight_test.go:22, planning/planning_test.go:17, acceptance/acceptance_test.go:26 all call t.Setenv("DACLI_AGENT", ""), and internal/cli/main_test.go TestMain os.Unsetenv(agentid.EnvVar) with a comment naming exactly this 'tool developing itself' hazard. Impact: DOGFOOD.md is the standard build mode, so the mandated Proof gate 'go test ./...' shows RED to every agent that runs it here, training agents to ignore a red suite or waste cycles diagnosing an environmental non-bug. Fix is test-only and pattern-established: add t.Setenv("DACLI_AGENT", "") in the cmdCatalog-driving test (or a TestMain clear for the package).
