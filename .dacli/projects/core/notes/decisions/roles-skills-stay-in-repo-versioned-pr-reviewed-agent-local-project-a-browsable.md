---
id: d-roles-skills-stay-in-repo-versioned-pr-reviewed-agent-local-project-a-browsable
kind: note
note_kind: decision
created: 2026-07-22T18:48:19Z
created_by: a-root
---
# Roles/skills stay in-repo (versioned + PR-reviewed + agent-local); project a browsable catalog to the wiki, add explicit versions
## Chose
Roles/skills stay in-repo (versioned + PR-reviewed + agent-local); project a browsable catalog to the wiki, add explicit versions
## Rejected
make the GitHub Wiki the primary editable store for roles
## Because
roles are permission config (grant/sandbox/kind) — changes need REVIEW (PR), must travel with the code, must be agent-readable locally at spawn, and keep typed frontmatter; a wiki gives none of those (direct edits, separate repo, network fetch, unstructured). Keep .dacli/ as source of truth (already git-versioned + reviewable + local); PROJECT a generated catalog to the wiki for humans; edits return via PR. Skills (knowledge) tolerate the wiki better than roles (config). Add explicit version: + a git-history changelog for clear versions
