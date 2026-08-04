---
id: 01KZ6ST6V9FM1758TQKMXEC3FA
kind: event
event_kind: finding
created: 2026-08-04T16:30:53Z
created_by: a-reviewer-664htb
about: "[[t-01KZ6S9FVG533ABEVXZBZZ7SC3]]"
origin: agent
applied: true
---
land: dacli pr --auto exits 0 with only a stderr note when auto-merge cannot be queued -- a headless agent reading exit code sees success while the PR is stranded open

Stage: LAND. cmdPR --auto (lifecycle.go:288-295): if 'gh pr merge --auto' fails (repo has no branch protection and 'Allow auto-merge' is off, or GitHub unreachable), dacli writes a stderr note ('note: auto-merge not queued for <branch> (PR stays open to merge)') and returns nil -- EXIT 0. stdout says nothing failed. The git_workflow.md brief tells the agent 'pr --auto queues GitHub native auto-merge so the PR lands itself the instant its required checks go green'; a HEADLESS agent (no human watching) that keys off exit code + stdout believes the task landed and exits, leaving the PR open forever with no one to merge it. Inconsistent with the sibling path: integrate --pr --auto (prIntegrateTask, lifecycle.go:~1239-1249) treats the same 'gh pr merge --auto' failure as a HARD error (non-zero) because '--auto asked GitHub to own the merge'. Two --auto surfaces disagree on whether an unqueueable auto-merge is fatal. Fix: on --auto failure, either exit non-zero, or print the stranded state to stdout so the agent can escalate. Not in backlog (083 is the feature completion note; the integrate-pr-cannot-advance finding is a different bug).
