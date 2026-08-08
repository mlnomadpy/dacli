---
id: f-092-complete-accept-all-force-reconciles-orphaned-tasks-ship-forwards-it
kind: note
note_kind: finding
created: 2026-07-23T10:34:56Z
created_by: a-c4n7ak99hj
about: [[092]]
severity: moderate
---
# 092 complete: accept --all --force reconciles orphaned tasks, ship forwards it
Commit d2fe2ae by a-c4n7ak99hj (fixer). ACCEPTANCE: (1) acceptAll (internal/features/acceptance/acceptance.go:145-183) gained a force bool param; when the acting identity is root and force is set, a proposed task owned by another (finished/orphaning) agent is adopted (owner rewritten to a-root, adoption logged) and closed instead of skipped — mirrors the existing single-ref override at acceptance.go:73-86. cmdAccept (acceptance.go:59-61) now passes f.Bool("force") through. ship.go's accept step (ship.go:98-108) now always shells 'accept --all --force' (and printPlan's dry-run line matches, ship.go:250-256) — safe unconditionally because accept only honors --force for root (agentid.RootID), so a non-root ship run is unaffected. (2) New test TestAcceptAllForceReconcilesOrphanedTask (acceptance_test.go) builds a task owned by a-deadchild with a pending proposal, asserts accept --all without --force leaves it open and accept --all --force closes it with owner adopted to a-root. Updated ship_test.go's exact-match assertion for the shelled accept args to 'accept --all --force'. docs/RUNTIMES.md accept/ship rows updated. go build ./... clean; go test ./internal/... all green. Box-checking is owner-only (only a-root) so filing as a finding — owner should verify and close via dacli task check/done + dacli merge --task 092.
