---
id: f-canonical-playbook-audit-exposed-four-stale-command-help-signatures
kind: note
note_kind: finding
created: 2026-08-19T12:08:22Z
created_by: a-maintainer-ebqr9f
about: "[[t-01M0CX031NDQ5PQ8VRX1PQNWXE]]"
severity: major
---
# Canonical playbook audit exposed four stale command help signatures
Before correction, current --help advertised queue next for the next scheduler, github push for branch push, and omitted supported project/dry-run arguments for github pull/sync. Reproduced with /tmp/dacli-current-bin next --help and push --help; guarded by docs/support_claims_test.go:41.
