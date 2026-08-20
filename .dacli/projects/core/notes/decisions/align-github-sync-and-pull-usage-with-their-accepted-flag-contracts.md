---
id: d-align-github-sync-and-pull-usage-with-their-accepted-flag-contracts
kind: note
note_kind: decision
created: 2026-08-19T12:42:21Z
created_by: a-fixer-00g3ry
about: "[[t-01M0AF65RDNBEX2SEF9JC5RTMZ]]"
github:
  issue: 720
  repo: mlnomadpy/dacli
---
# Align github sync and pull Usage with their accepted flag contracts
## Chose
Align github sync and pull Usage with their accepted flag contracts
## Rejected
Leave github sync as a bare command and document only a separate preview flow
## Because
The aggregated command table drives both CLI help and MCP help, so its Usage must name flags that cmdSync forwards to cmdPush and cmdPull.
