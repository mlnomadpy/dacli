---
id: t-01KZ6BM3FMSAKQWAJFTJY8CFFG
kind: task
created: 2026-08-04T12:22:53Z
created_by: a-maintainer-8zn6wb
owner: a-maintainer-8zn6wb
priority: should
---
# catalog test package leaks DACLI_AGENT — add the env-clear other command tests use so the dogfood suite is green
## Acceptance
- [ ] Running the suite from within a dacli agent session (DACLI_AGENT set) is green: 'go test ./internal/features/catalog/' no longer fails TestCatalogRefusesRatherThanWritingAnEmptyRoster with 'agent token not recognized'
- [ ] The catalog test that drives cmdCatalog clears DACLI_AGENT the way sibling command tests do — t.Setenv("DACLI_AGENT", "") (see insight_test.go:22, planning_test.go:17, acceptance_test.go:26) or a package TestMain os.Unsetenv(agentid.EnvVar) (see cli/main_test.go) — with a comment naming the dogfood 'tool developing itself' reason
- [ ] Fix is test-only (no product/source change); 'go build ./... && go test ./... && go vet ./... && gofmt -l internal/' all clean when run under a dacli agent
## Log
- 2026-08-04T12:30:58Z claimed by a-maintainer-abgsm8
