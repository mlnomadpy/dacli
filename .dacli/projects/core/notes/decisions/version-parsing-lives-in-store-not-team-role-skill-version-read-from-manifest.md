---
id: d-version-parsing-lives-in-store-not-team-role-skill-version-read-from-manifest
kind: note
note_kind: decision
created: 2026-07-22T20:15:24Z
created_by: a-64p4b0yykq
about: [[068]]
---
# Version parsing lives in store, not team.Role; skill version read from manifest frontmatter directly
## Chose
Version parsing lives in store, not team.Role; skill version read from manifest frontmatter directly
## Rejected
add a Version field to team.Role / skills.Skill structs
## Because
internal/team and internal/skills are outside 068's STRICT scope (store, teamops, skillforge only); store.FileVersion(path) reads the version: frontmatter for both objects, and store.FileChangelog runs git log on the manifest file, keeping all new code inside the three allowed packages
