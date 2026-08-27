---
id: 01M106R93YG14NX4Q2WM795HCM
kind: event
schema_version: 1
event_kind: commit
created: 2026-08-26T23:33:44Z
created_by: a-root
about: "[[t-01M0ZCAQ05J2H9VHB4BA9YTQGD]]"
origin: agent
applied: true
checksum: sha256:d93ec268ef6ec90e03e002a2f238553c3d60934b0f24666ea62fd29260e23cdf
---
5e56f70 fix: honor project landing base during acceptance

Mutation: replacing configured dev with repository master makes TestAcceptUsesConfiguredLandingBaseForConfirmedMerge fail with NOT in master.
role: root
