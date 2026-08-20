---
id: f-dependency-cycle-mutation-is-caught-by-the-focused-store-regression
kind: note
note_kind: finding
created: 2026-08-20T08:32:17Z
created_by: a-maintainer-d7gr0n
about: "[[t-01M0CZANAQKP50AWEN2C6C8VXR]]"
severity: major
---
# Dependency cycle mutation is caught by the focused store regression
Temporarily replacing the spm.ComputeCPM error guard in internal/store/dependency.go with an ignored result made GOCACHE=/tmp/dacli-go-cache-479 go test ./internal/store -run '^TestDependencyChangeValidationFailuresDoNotWrite/cycle$' fail at internal/store/dependency_test.go:80 with 'invalid dependency change succeeded'; restoring the guard returns green.
