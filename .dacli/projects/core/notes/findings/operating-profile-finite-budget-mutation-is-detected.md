---
id: f-operating-profile-finite-budget-mutation-is-detected
kind: note
note_kind: finding
created: 2026-08-19T15:21:12Z
created_by: a-maintainer-3necr2
about: "[[t-01M0CX03Q4A1BM8JD9YQBCNGV0]]"
severity: major
---
# Operating profile finite-budget mutation is detected
Changing defaultProfile RollingTokens from 240000 to 0 made GOCACHE=/tmp/dacli-go-cache-477 go test ./internal/features/orchestration -run '^TestOperatingProfileGoldenDefaultsAreFiniteAndReleaseIsOff$' fail at profile_test.go:35 with 'task has an unbounded default'; restoring the ceiling returns green.
