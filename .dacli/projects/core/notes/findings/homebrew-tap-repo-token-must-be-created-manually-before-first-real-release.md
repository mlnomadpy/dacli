---
id: f-homebrew-tap-repo-token-must-be-created-manually-before-first-real-release
kind: note
note_kind: finding
created: 2026-07-23T18:57:24Z
created_by: a-jze25mtf8b
about: [[120]]
severity: moderate
---
# Homebrew tap repo + token must be created manually before first real release
goreleaser brews (.goreleaser.yaml:58-71) pushes the generated formula to github.com/mlnomadpy/homebrew-tap on every 'v*' tag push, authenticated via the HOMEBREW_TAP_GITHUB_TOKEN repo secret (used in .github/workflows/release.yml). Neither the tap repo nor the secret exists yet -- this is a one-time manual setup step for the owner before the first real tag; until then the release workflow's brew-formula-push step will fail (binaries/checksums/GitHub release still succeed since brews runs after those). No tag was pushed and no release was triggered by this task.
