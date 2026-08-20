---
id: d-pass-caller-cwd-explicitly-into-verification-execution-and-git-lookup
kind: note
note_kind: decision
created: 2026-08-19T13:19:26Z
created_by: a-fixer-z2ed21
about: "[[t-01M0AGCX64GETGKKBBJK5SKG7D]]"
github:
  issue: 740
  repo: mlnomadpy/dacli
---
# Pass caller Cwd explicitly into verification execution and Git lookup
## Chose
Pass caller Cwd explicitly into verification execution and Git lookup
## Rejected
Keep RunVerification derived from workspace.Workspace
## Because
workspace.Find intentionally shares the main .dacli root across linked worktrees, while a verification must bind evidence to the caller checkout.
