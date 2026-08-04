---
id: d-223-release-is-rw-gated-but-not-disclosure-gated-ship-release-requires-push-and
kind: note
note_kind: decision
created: 2026-08-04T12:12:45Z
created_by: a-maintainer-prf0dg
about: "[[223]]"
---
# 223: release is rw-gated but NOT disclosure-gated; ship --release requires --push and refuses --pr
## Chose
223: release is rw-gated but NOT disclosure-gated; ship --release requires --push and refuses --pr
## Rejected
Running github release through disclosureGate like github push, and letting ship --release run in --pr mode / without --push
## Because
disclosureGate exists because push MIRRORS workspace finding/decision notes (internal reasoning not otherwise in the repo) to a public tracker. gh --generate-notes assembles notes from the repo's OWN merged-PR/commit history, already at the repo's visibility, so a release discloses nothing new — gating it would be cargo-cult. It still needs rw (Method axiom 4: writes to a remote). ship --release requires --push because the release tags the REMOTE state (local-merge without push leaves origin/into stale → a record that lies), and refuses --pr because PR-first merges to the target asynchronously on GitHub's clock (an --auto PR merges only on CI green), so a release cut immediately could tag the target before the wave's PRs land. Preconditions validated up front so a wave is never half-shipped then refused.
