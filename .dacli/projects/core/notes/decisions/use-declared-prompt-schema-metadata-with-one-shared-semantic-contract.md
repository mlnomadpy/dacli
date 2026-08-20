---
id: d-use-declared-prompt-schema-metadata-with-one-shared-semantic-contract
kind: note
note_kind: decision
created: 2026-08-19T14:29:23Z
created_by: a-maintainer-anf4d3
about: "[[t-01M0CX03CC0N95X4M5ESKRP2E6]]"
github:
  issue: 750
  repo: mlnomadpy/dacli
---
# Use declared prompt schema metadata with one shared semantic contract
## Chose
Use declared prompt schema metadata with one shared semantic contract
## Rejected
Version copied provider-specific prompt variants or silently accept legacy overrides
## Because
A stripped declaration preserves emitted compatibility while failing closed on semantic-version drift; one contract prevents lifecycle rules diverging across runtime adapters
