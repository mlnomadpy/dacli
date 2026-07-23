---
id: d-filed-the-unbounded-git-in-the-loop-deadline-fix-as-the-single-highest-value
kind: note
note_kind: decision
created: 2026-07-23T12:42:09Z
created_by: a-g3ya9r93e3
about: [[084]]
---
# Filed the unbounded-git-in-the-loop deadline fix as the single highest-value change
## Chose
Filed the unbounded-git-in-the-loop deadline fix as the single highest-value change
## Rejected
the loop's lowest-seq batching (already task 103), force-close of failed spawns (already task 102), or the lower-risk local-git callers store/version.go and skills.go git clone
## Because
102 and 103 already cover the two obvious loop bugs, so re-filing them adds nothing. Among the remaining unbounded-git violators, driver.git is uniquely dangerous: it runs a NETWORK fetch every cycle inside the always-on perpetual loop (orchestration.go:402,268), so a single wedged fetch freezes the whole autonomous machine with no window/thrash/stop-file recovery — strictly worse than a wrong decision. store/version.go and skills.go are local/one-shot and lower risk, so they ride along as secondary acceptance rather than driving a separate task.
