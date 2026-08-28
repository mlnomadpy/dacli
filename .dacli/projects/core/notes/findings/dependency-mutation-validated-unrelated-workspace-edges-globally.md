---
id: f-dependency-mutation-validated-unrelated-workspace-edges-globally
kind: note
note_kind: finding
created: 2026-08-28T00:18:36Z
created_by: a-maintainer-z0nyk9
about: "[[t-01M1068M8HJ9G8XCXMEMVE2V8D]]"
severity: major
---
# Dependency mutation validated unrelated workspace edges globally
internal/store/dependency.go built one global TaskIndex and iterated every stored task edge, so an unrelated q/001 legacy ref failed as globally ambiguous against p/001 before a qualified p edge could be saved. Regression reproduced with: GOCACHE=/tmp/dacli-go-cache go test ./internal/store -run TestDependencyChangeIgnoresUnrelatedAmbiguousLegacyRefs -count=1.
