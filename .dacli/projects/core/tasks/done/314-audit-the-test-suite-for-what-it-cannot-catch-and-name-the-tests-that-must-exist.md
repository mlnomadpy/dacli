---
id: t-01KZNYJ7P67QTZ0B65JPCZN47C
kind: task
created: 2026-08-10T13:42:31Z
created_by: a-root
owner: a-root
priority: must
estimate: "{optimistic: 3, probable: 5, pessimistic: 8}"
---
# Audit the test suite for what it cannot catch, and name the tests that must exist
## So that
the suite proves behaviour rather than reporting green, after a session where two tests passed while measuring nothing
## Acceptance
- [x] every package is assessed for what a passing run does NOT prove, with the specific untested behaviour named
- [x] tests that pass for the wrong reason are identified by driving them, not by reading them — the flag-rejection test passed with zero failures while asserting nothing
- [x] the report separates missing coverage from weak assertions, because they need different fixes
## Log
- 2026-08-10T13:43:22Z claimed by a-go-auditor-c2r8as
- 2026-08-10T13:57:44Z finding by a-go-auditor-c2r8as: propose:done close-path test manufactures a precondition the real actor cannot reach (event 01KZNYVPRBR9FZ3XY4JX77HFJS)
- 2026-08-10T13:57:44Z finding by a-go-auditor-c2r8as: CORRECTION: my sync_test 'unreachable precondition' claim is wrong; the real gap is integration-level (event 01KZNZ21617Q85SGH414GGTMH0)
- 2026-08-10T13:57:44Z finding by a-go-auditor-c2r8as: Malformed compound assertion silences the exact regression it names (guards_test.go:365) (event 01KZNZ2DHKN4BQGM97NJZFR299)
- 2026-08-10T13:57:44Z finding by a-go-auditor-c2r8as: Test-suite audit synthesis: what a green run does NOT prove, missing-coverage vs weak-assertions (event 01KZNZ33M5JW2VJ5CA50VYCCS8)
- 2026-08-10T14:13:28Z accepted by a-root
- 2026-08-10T14:13:28Z closed WITHOUT verification — no --verify command was given
- 2026-08-10T14:13:28Z deliverable: dacli/314-audit-the-test-suite-for-what-it-cannot-catch-and-name-the-tests-that-must-exist is merged into trunk
- 2026-08-10T14:13:28Z completed by a-root
