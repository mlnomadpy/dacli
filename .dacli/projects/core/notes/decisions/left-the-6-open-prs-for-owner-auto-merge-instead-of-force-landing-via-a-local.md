---
id: d-left-the-6-open-prs-for-owner-auto-merge-instead-of-force-landing-via-a-local
kind: note
note_kind: decision
created: 2026-08-04T11:06:18Z
created_by: a-integrator-ydcqew
about: "[[250]]"
---
# Left the 6 open PRs for owner/auto-merge instead of force-landing via a local merge+direct push to main
## Chose
Left the 6 open PRs for owner/auto-merge instead of force-landing via a local merge+direct push to main
## Rejected
dacli merge/integrate (no --pr) to locally merge each branch into main, then push origin main
## Because
mergeTask (lifecycle.go:889) is a raw local git merge with NO CI gate; landing it requires a direct push to the protected public main, which bypasses the 'merges only PRs whose checks pass' gate the task demands and violates 'never merge red' (I cannot read gh pr checks in this sandbox) and 'do not push'. The check-gated gh path (integrate --pr) is unavailable because it aborts on the already-open PRs. Safe outcome: 206 self-lands via its queued auto-merge; the rest are reported blocked with the tool-defect finding and issue #294 for the owner to enable auto-merge or run integrate from a gh-capable host.
