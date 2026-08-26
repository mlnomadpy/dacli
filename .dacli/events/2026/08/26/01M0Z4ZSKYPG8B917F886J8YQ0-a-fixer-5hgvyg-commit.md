---
id: 01M0Z4ZSKYPG8B917F886J8YQ0
kind: event
schema_version: 1
event_kind: commit
created: 2026-08-26T13:43:38Z
created_by: a-fixer-5hgvyg
about: "[[t-01M0D4SN9N7MP3A02J76JZ32KW]]"
origin: agent
applied: true
checksum: sha256:e5f9e4e9aafe217bf71603316c8c1924e46682220bb51c339eb3e34dd0ff88b1
---
3ed6150 t-01M0D4SN9N7MP3A02J76JZ32KW: join detached test guardian before TempDir cleanup

Wait for the guardian's runtime-exit marker after recorder completion so its final write cannot race test cleanup. Mutation: removing the guardian join makes TestAwaitDetachedCompletionWaitsForGuardianFinalWrite fail: detached completion returned before guardian's final TempDir write.
role: fixer
