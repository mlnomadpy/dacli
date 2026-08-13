---
id: d-use-one-credential-free-fixture-registry-as-contract-and-matrix-source
kind: note
note_kind: decision
created: 2026-08-13T09:49:26Z
created_by: a-codex-maintainer-2hqkmd
about: "[[404]]"
---
# Use one credential-free fixture registry as contract and matrix source
## Chose
Use one credential-free fixture registry as contract and matrix source
## Rejected
Maintain vendor-specific behavioral tests and a hand-edited support table
## Because
The shared registry drives every adapter through identical prompt, model, result, usage, timeout, cancellation, sandbox, write, and exit assertions; the published matrix is byte-checked against that same executable source.
