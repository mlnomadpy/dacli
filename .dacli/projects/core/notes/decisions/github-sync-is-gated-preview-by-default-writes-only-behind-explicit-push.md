---
id: d-github-sync-is-gated-preview-by-default-writes-only-behind-explicit-push
kind: note
note_kind: decision
created: 2026-07-22T16:27:34Z
created_by: a-root
---
# GitHub sync is gated: preview by default, writes only behind explicit --push; targets the configured (private) repo
## Chose
GitHub sync is gated: preview by default, writes only behind explicit --push; targets the configured (private) repo
## Rejected
auto-sync agent-generated decisions/findings to GitHub on every ship
## Because
materializing agent content to GitHub is a disclosure action; the operator must trigger it. Best decision: dacli builds the projection and shows a dry-run diff of what WOULD be created/closed; actual issue/discussion/PR writes require an explicit --push flag, exactly like dacli ship --push. Capability delivered, disclosure boundary respected.
