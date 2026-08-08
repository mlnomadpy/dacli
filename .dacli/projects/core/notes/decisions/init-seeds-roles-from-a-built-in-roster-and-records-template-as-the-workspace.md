---
id: d-init-seeds-roles-from-a-built-in-roster-and-records-template-as-the-workspace
kind: note
note_kind: decision
created: 2026-07-22T16:22:26Z
created_by: a-sfa41hsara
about: [[053]]
---
# init seeds roles from a built-in roster and records --template as the workspace default that project add falls back to
## Chose
init seeds roles from a built-in roster and records --template as the workspace default that project add falls back to
## Rejected
vendor the template file only, or drop the flags from the Brief and docs
## Because
the spec (TEMPLATES.md §8, TEAM.md §2, WALKTHROUGH.md §1) advertises init --template/--roster as first-run seeding; making --roster create real role files and --template a validated, honored default gives both flags a genuine mechanical effect instead of silently accepting them
