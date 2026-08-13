---
id: t-01KZY9YGP5XDD748GEGQG2ACZ2
kind: task
created: 2026-08-13T19:35:23Z
created_by: a-codex-loop-auditor-hxqjcg
owner: a-root
priority: must
estimate: "{optimistic: 2, probable: 3, pessimistic: 5}"
parent: "[[t-01KZXXZPBXT332W00RP94HTR2K]]"
github:
  issue: 603
  repo: mlnomadpy/dacli
---
# Fix tagged release SBOM generation by installing Syft
## So that
a manually pushed v* tag executes the already-verified release path instead of failing after archives are built
## Acceptance
- [x] .github/workflows/release.yml installs a pinned Syft distribution before goreleaser runs release --clean
- [x] .github/workflows/contract_test.go fails when the release workflow omits or reorders the Syft prerequisite relative to GoReleaser
- [x] a non-publishing snapshot run produces all six configured archives, an SBOM per archive, and a checksums.txt that verifies without creating or pushing a tag
## Log
- 2026-08-13T19:41:44Z claimed by a-codex-maintainer-9gwn2s
- 2026-08-13T19:53:14Z adopted by a-root (owner a-codex-loop-auditor-hxqjcg orphaned)
- 2026-08-13T19:53:14Z accepted by a-root
- 2026-08-13T19:53:14Z verified by `GOCACHE=/private/tmp/dacli-429-main-gocache go test ./.github/workflows` (exit 0) in branch main at db66a1b — proves that tree builds, not that the work is in trunk
- 2026-08-13T19:53:14Z deliverable: dacli/429-fix-tagged-release-sbom-generation-by-installing-syft is merged into main
- 2026-08-13T19:53:14Z completed by a-root
- 2026-08-13T20:09:37Z a-root: PR opened: https://github.com/mlnomadpy/dacli/pull/610 (event 01KZYAKWA4VN819XT8RG9D70QR)
