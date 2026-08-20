---
id: f-committed-vcs-test-catches-restoration-of-the-truncated-commit-synopsis
kind: note
note_kind: finding
created: 2026-08-19T14:14:32Z
created_by: a-fixer-gcha7z
about: "[[t-01M0D2KPCZ5PEFXJS4B0J59Z5C]]"
severity: major
---
# Committed VCS test catches restoration of the truncated commit synopsis
Final mutation evidence: changing the commit command table Usage back to the truncated literal made GOCACHE=/tmp/dacli-go-cache-483 go test ./internal/features/vcs -run ^TestCommitUsageMatchesCommandTable$ fail at internal/features/vcs/commit_test.go:35.
