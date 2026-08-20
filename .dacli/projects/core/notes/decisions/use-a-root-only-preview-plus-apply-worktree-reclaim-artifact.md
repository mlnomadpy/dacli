---
id: d-use-a-root-only-preview-plus-apply-worktree-reclaim-artifact
kind: note
note_kind: decision
created: 2026-08-19T13:23:06Z
created_by: a-maintainer-a5y9am
about: "[[t-01M0AGCX29Q047FZHKNG3YV0WC]]"
github:
  issue: 741
  repo: mlnomadpy/dacli
---
# Use a root-only preview-plus-apply worktree reclaim artifact
## Chose
Use a root-only preview-plus-apply worktree reclaim artifact
## Rejected
Mutate proc.txt ownership or accept an implicit commit-time takeover
## Because
A separate atomic worktree-transfer.txt preserves historical run identity/outcome, makes the new owner and claims durable, and requires root to inspect exact branch, dirty paths, prior run/owner, and claims before mutation.
