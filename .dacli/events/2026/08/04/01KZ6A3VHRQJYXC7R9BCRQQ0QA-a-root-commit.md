---
id: 01KZ6A3VHRQJYXC7R9BCRQQ0QA
kind: event
event_kind: commit
created: 2026-08-04T11:56:32Z
created_by: a-root
origin: agent
applied: false
---
fab0b7c reconcile the two collided seqs 251 found, and file 260

`dacli doctor` grew a collided-seq check in 251 and immediately named
the wreckage already in the repo: seq 250 claimed by both the
grant/runtime task and the merge-the-wave task, and seq 251 by both the
seq-allocation task and the token-leak test fix. Either one made
`dacli 250` ambiguous — which is how `spawn --task 251` failed to
resolve earlier today.

The closed duplicates are renumbered (251-testid -> 258,
250-merge-the-wave -> 259) so the live tasks keep the numbers people
have been referring to. The stale active/ copy of 251-testid, left when
its done/ move landed through a separate branch, is removed.

doctor is down to one finding, and that one predates today.

260 is the question this wave raised and did not answer: `dacli accept`
run inside a worktree wrote to the MAIN workspace, not the worktree's
own .dacli, so every agent in a wave shares one record store. It works —
247's lock is what makes it safe — but nothing says whether it is
intended, and an agent's record of its own work arguably belongs on its
own branch.
role: root
