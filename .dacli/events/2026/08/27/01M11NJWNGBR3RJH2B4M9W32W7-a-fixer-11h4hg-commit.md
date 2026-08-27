---
id: 01M11NJWNGBR3RJH2B4M9W32W7
kind: event
schema_version: 1
event_kind: commit
created: 2026-08-27T13:12:10Z
created_by: a-fixer-11h4hg
about: "[[t-01KZYREM23X0ADW8MDV26C1H9A]]"
origin: agent
applied: true
checksum: sha256:9f4791bda0dd2571b2dd52bf0ae4463e61938f07a0f629890fa5c6049a03330f
---
b3d5a4b t-01KZYREM23X0ADW8MDV26C1H9A: project acceptance before dry-run explicit ship wave

Dry-run previously rejected a proposed active task before the accept transition that real ship performs. Project static acceptance eligibility without executing verification or mutating the workspace, so its explicit integrate window matches real ship. Mutation proof: forcing the preview to skip this transition made TestShipDryRunExplicitProposedActiveWindowProjectsAccept fail with the original active/not-done refusal.
role: fixer
