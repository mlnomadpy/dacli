---
id: 01KZYKT6DRBMJRQC5PQYE130QZ
kind: event
schema_version: 1
event_kind: commit
created: 2026-08-13T22:27:47Z
created_by: a-fixer-btcg7r
about: "[[t-01KZYHZTP8NJVY9TYM7S2ANJ38]]"
origin: agent
applied: true
checksum: sha256:0c774179887919bcb0232d6bbe69b1662a43e495576d68f2ad0d19d9f7f3affb
---
a43daa7 437: deduplicate semantic GitHub mirror records

Canonicalize near-duplicate decision and finding titles before planning or remote writes, retain all distinct evidence, and preserve every source marker for partial-push recovery.

Mutation proof: before canonicalNoteFiles existed, GOCACHE=/private/tmp/dacli-437-go-cache go test ./internal/features/ghmirror -run TestCanonicalNoteFiles -count=1 failed with undefined: canonicalNoteFiles.
role: fixer
