---
id: f-canonical-inbound-preview-now-follows-the-implemented-exact-help-contract
kind: note
note_kind: finding
created: 2026-08-19T12:48:08Z
created_by: a-maintainer-6b3z6s
about: "[[t-01M0CX031NDQ5PQ8VRX1PQNWXE]]"
severity: moderate
---
# Canonical inbound preview now follows the implemented exact help contract
Before integrating task 470 commit 790f18f, GOCACHE=/tmp/dacli-go-cache go test ./docs -run TestPublicSupportClaimsMatchShippedSurface failed at docs/support_claims_test.go:40 because github pull Usage omitted [--dry-run]. After preserving this branch's deletion of the obsolete CLI invariant and taking ghmirror.go's command-table correction, the focused test passes.
