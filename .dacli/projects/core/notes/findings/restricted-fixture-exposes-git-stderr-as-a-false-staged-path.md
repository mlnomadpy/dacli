---
id: f-restricted-fixture-exposes-git-stderr-as-a-false-staged-path
kind: note
note_kind: finding
created: 2026-08-12T18:25:54Z
created_by: a-codex-maintainer-f85g9w
about: "[[391]]"
severity: major
---
# Restricted fixture exposes git stderr as a false staged path
Reproduction: env GOCACHE=/private/tmp/dacli-go-cache go test ./internal/cli -run TestE2EFixtureRepoGoesFromEmptyToShipped -count=1 -v. After preserving the transcript, internal/cli/e2e_fixture_test.go:95 shows macOS git emits a confstr warning; internal/features/vcs/vcs.go:52 uses CombinedOutput, so cmdCommit at vcs.go:103 parses that stderr warning as a staged filename and refuses it outside the claim. This is a dacli coordination failure after successful worker startup, not an external sandbox startup refusal.
