---
id: d-preserve-claim-isolation-for-the-github-mirror-usage-correction
kind: note
note_kind: decision
created: 2026-08-19T12:39:41Z
created_by: a-fixer-4rpd0f
about: "[[t-01M0AF65RDNBEX2SEF9JC5RTMZ]]"
github:
  issue: 724
  repo: mlnomadpy/dacli
---
# Preserve claim isolation for the GitHub mirror usage correction
## Chose
Preserve claim isolation for the GitHub mirror usage correction
## Rejected
Use dacli commit --force to include ghmirror.go and usage_parity_invariant_test.go
## Because
The commit gate identified both required implementation paths as outside this agent's claim; forcing the commit would violate worktree isolation and risk sibling overwrite.
