---
id: d-244-filed-task-262-catalog-test-leaks-dacli-agent-breaking-the-dogfood-suite-as
kind: note
note_kind: decision
created: 2026-08-04T12:23:09Z
created_by: a-maintainer-8zn6wb
about: "[[244]]"
---
# 244: filed task 262 (catalog test leaks DACLI_AGENT, breaking the dogfood suite) as the single highest-value evidence-based change
## Chose
244: filed task 262 (catalog test leaks DACLI_AGENT, breaking the dogfood suite) as the single highest-value evidence-based change
## Rejected
Filing a fresh unverified correctness lead from the sibling findings (016/021/026/027/028 completions), or re-filing an already-open item (165/200/224/260/261), or filing the broader 'harden ALL test packages against DACLI_AGENT leak' class
## Because
A live FAILING check outranks every unverified lead: 'go test ./...' run as a dacli agent FAILS only in internal/features/catalog (TestCatalogRefusesRatherThanWritingAnEmptyRoster, rosterwipe_test.go:41 'agent token not recognized'), and 'go test -exec env -u DACLI_AGENT ./internal/features/catalog/' PASSES — proving the sole cause is the env leak, not a code defect. The sibling f-016/f-021 completions are unverified maintainer self-reports on un-integrated branches (not on main, not my scope to verify here); the open tasks 165/200/224/260/261 are distinct and already queued. A global 'harden all packages' task is speculative — catalog is the ONLY package that actually fails, and insight/planning/acceptance/cli already clear the env, so the fix is a single pattern-matched test edit, not a sweep. This defect corrupts the record in the worst documented way: DOGFOOD.md is the standard build mode, so the Proof gate the method MANDATES shows RED to every agent, training them to ignore a red suite. task add refused --owner (strict-flag rejection now live) and accepted the title as distinct scope (no exit-3 near-dup).
