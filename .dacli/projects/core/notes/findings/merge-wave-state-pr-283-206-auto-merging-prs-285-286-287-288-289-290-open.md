---
id: f-merge-wave-state-pr-283-206-auto-merging-prs-285-286-287-288-289-290-open
kind: note
note_kind: finding
created: 2026-08-04T11:06:01Z
created_by: a-integrator-ydcqew
about: "[[250]]"
severity: moderate
---
# Merge-wave state: PR #283 (206) auto-merging; PRs #285/#286/#287/#288/#289/#290 open awaiting merge w/o auto-merge; PR #46 (085) closed-orphaned
Enumerated the wave via dacli pr status + git branch -r --no-merged origin/main. Every diff verified in-scope for its task (git diff --stat origin/main...branch). State: (1) task 206 raise-coverage / PR #283 -> auto-merge QUEUED, self-lands on green CI, no action needed. (2) tasks 205(#285 ghmirror), 210(#289 acceptance), 215(#288 worktree-path), 243(#286 int-flags), 247(#290 acquireseqlock), 248(#287 lesson-matcher) -> PRs OPEN, 'awaiting merge', auto-merge NOT queued, so they will NOT self-land; blocked by the integrate/pr create-first defect [[dacli-integrate-pr-pr-auto-cannot-advance-an-already-open-pr-gh-pr-create-runs]] and by gh being unavailable in-sandbox. (3) task 085 (the only one of these in the done column) / PR #46 -> CLOSED without merging = orphaned (task-157/160 pattern); its branch dacli/085 is unmerged. CI green/red status of the 6 open PRs could NOT be read (gh pr checks needs gh, not in sandbox), so none should be force-landed via a local merge+direct push, which would also bypass the CI gate and push to protected main. Owner action: enable auto-merge on the 6 (gh pr merge <branch> --auto --merge) or run integrate from a host with gh; decide 085's fate (its feature may be superseded on main).
