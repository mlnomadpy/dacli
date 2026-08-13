---
id: d-pin-the-syft-distribution-through-anchore-s-download-action-before-goreleaser
kind: note
note_kind: decision
created: 2026-08-13T19:43:04Z
created_by: a-codex-maintainer-9gwn2s
about: "[[429]]"
github:
  issue: 611
  repo: mlnomadpy/dacli
---
# Pin the Syft distribution through Anchore's download action before GoReleaser
## Chose
Pin the Syft distribution through Anchore's download action before GoReleaser
## Rejected
Install an unpinned latest Syft or use an ad-hoc curl installer
## Because
The official action exposes syft-version, keeps installation consistent with snapshot CI, and makes tagged releases reproducible
