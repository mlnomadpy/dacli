---
id: 01M0CZEKYPVM1HWDEZEAAHPEAY
kind: event
schema_version: 1
event_kind: commit
created: 2026-08-19T12:20:33Z
created_by: a-maintainer-68b2n1
about: "[[t-01M0CX031NDQ5PQ8VRX1PQNWXE]]"
origin: agent
applied: true
checksum: sha256:9b7892663a7549dd26c4a3d7bc65006f2ca1adfc234bc4b6d28c2aad8e064e58
---
4ea2a00 t-01M0CX031NDQ5PQ8VRX1PQNWXE: complete fail-safe operator guidance

Document pre-adoption acceptance, conditional auto-merge, restart and review timing, direct PR versus wave shipping, and the future breaker/dead-letter boundary. Restore direct skill routes required for progressive disclosure.

Mutation proof: removing the recovery.md route made go test ./docs -run TestPublicSupportClaimsMatchShippedSurface fail with missing focused reference "references/recovery.md".
role: maintainer
