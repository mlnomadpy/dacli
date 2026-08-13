---
id: f-tagged-release-workflow-cannot-generate-configured-sboms-because-syft-is-absent
kind: note
note_kind: finding
created: 2026-08-13T19:35:23Z
created_by: a-codex-loop-auditor-hxqjcg
about: "[[426]]"
severity: major
---
# Tagged release workflow cannot generate configured SBOMs because Syft is absent
.github/workflows/release.yml:46-52 invokes goreleaser release --clean without installing Syft, while .goreleaser.yaml:44-53 configures one Syft-generated SBOM per archive. Reproduction on current HEAD with Syft absent: GOCACHE=/private/tmp/dacli-426-gocache goreleaser release --snapshot --clean builds and archives all six targets, then exits 1 at software bill of materials with exec: "syft": executable file not found in PATH. The CI snapshot avoids this only because .github/workflows/ci.yml installs anchore/sbom-action/download-syft before scripts/verify-release-artifacts.sh; that step is absent from the actual tag workflow. Suspected cause: task 356 hardened the snapshot CI path but did not mirror its Syft prerequisite into the publish path. Manual recovery would be to install Syft before rerunning the tag job. Duplicate check: rg across all core tasks/notes for release.yml+syft/missing syft found no match; open/active/blocked lists contain no equivalent implementation task.
