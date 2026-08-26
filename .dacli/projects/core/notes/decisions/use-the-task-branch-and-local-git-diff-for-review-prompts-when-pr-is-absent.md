---
id: d-use-the-task-branch-and-local-git-diff-for-review-prompts-when-pr-is-absent
kind: note
note_kind: decision
created: 2026-08-26T13:15:04Z
created_by: a-fixer-s3nggb
about: "[[t-01M0CZANK00P2B5XY6TVJNAWCK]]"
---
# Use the task branch and local git diff for review prompts when --pr is absent
## Chose
Use the task branch and local git diff for review prompts when --pr is absent
## Rejected
Always resolve a GitHub PR through gh
## Because
Local landing has a canonical task branch but may intentionally have no GitHub PR or gh access.
