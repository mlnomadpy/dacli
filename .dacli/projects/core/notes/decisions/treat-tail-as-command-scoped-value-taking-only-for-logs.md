---
id: d-treat-tail-as-command-scoped-value-taking-only-for-logs
kind: note
note_kind: decision
created: 2026-08-27T13:08:37Z
created_by: a-fixer-0hn8n8
about: "[[t-01M0D3MKRKCHSX8P51HRDF0HQX]]"
github:
  issue: 831
  repo: mlnomadpy/dacli
---
# Treat --tail as command-scoped value-taking only for logs
## Chose
Treat --tail as command-scoped value-taking only for logs
## Rejected
Keep tail in the parser's global always-boolean set
## Because
logs documents --tail N, while agents uses bare --tail; a global classification cannot represent both forms.
