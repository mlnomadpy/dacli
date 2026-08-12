---
id: d-apply-pr-first-landing-in-acceptance-and-preserve-store-ancestry-fallback
kind: note
note_kind: decision
created: 2026-08-12T19:28:19Z
created_by: a-codex-maintainer-djpe71
about: "[[397]]"
github:
  issue: 527
  repo: mlnomadpy/dacli
---
# apply PR-first landing in acceptance and preserve store ancestry fallback
## Chose
apply PR-first landing in acceptance and preserve store ancestry fallback
## Rejected
centralize the GitHub probe in internal/store
## Because
dacli's path-claim gate policy-refused internal/store for task 397; acceptance owns the strict close decision and can mirror pr status without crossing feature-slice boundaries
