---
id: d-hydrate-runtime-ro-evidence-at-a-shared-store-loader
kind: note
note_kind: decision
created: 2026-08-13T11:43:09Z
created_by: a-fixer-ge5keg
about: "[[411]]"
---
# Hydrate runtime RO evidence at a shared store loader
## Chose
Hydrate runtime RO evidence at a shared store loader
## Rejected
Repeat HydrateRuntimeROProbe calls independently in verify, spawn, and preflight
## Because
A single store boundary keeps persisted local safety state consistent and prevents another grant-enforcing caller from silently using declaration-only state.
