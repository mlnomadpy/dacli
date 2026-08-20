---
id: f-worktree-reclaim-cannot-make-the-full-suite-green-within-this-agent-claim
kind: note
note_kind: finding
created: 2026-08-19T13:31:12Z
created_by: a-maintainer-a5y9am
about: "[[t-01M0AGCX29Q047FZHKNG3YV0WC]]"
severity: major
---
# Worktree reclaim cannot make the full suite green within this agent claim
go test ./... fails internal/cli TestEveryCommandDeclaresItsCapability because worktree reclaim must be added to capability_invariant_test.go, and TestTrustDocListsEveryMutatingCommand because docs/TRUST.md must name it. This run's proc.txt claims only internal/features/vcs,internal/gitx, so editing or force-committing either required path would violate slice isolation. Focused vcs and race tests pass; build/vet/gofmt pass; golangci-lint is unavailable.
