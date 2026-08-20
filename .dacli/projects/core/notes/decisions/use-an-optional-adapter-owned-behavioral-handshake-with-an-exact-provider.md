---
id: d-use-an-optional-adapter-owned-behavioral-handshake-with-an-exact-provider
kind: note
note_kind: decision
created: 2026-08-20T09:15:01Z
created_by: a-maintainer-dxj9ch
about: "[[t-01M0F3795JGCAG6ZS3XVAGNS2J]]"
github:
  issue: 761
  repo: mlnomadpy/dacli
---
# Use an optional adapter-owned behavioral handshake with an exact provider-neutral cache key
## Chose
Use an optional adapter-owned behavioral handshake with an exact provider-neutral cache key
## Rejected
Treat the existing sandbox flag probe as launch evidence or parse Codex failures in spawn
## Because
Issue #746 occurs after sandbox declaration validation; keeping argv and prose classification in the adapter seam lets spawn consume structured layers without vendor branches, while unsupported adapters preserve their declared behavior
