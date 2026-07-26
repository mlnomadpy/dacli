---
id: d-114-defer-the-self-pr-record-push-until-no-branch-this-loop-opened-is-still-on
kind: note
note_kind: decision
created: 2026-07-26T21:52:24Z
created_by: a-hp40s2ycrb
about: [[114]]
---
# 114: defer the self-PR record push until no branch this loop opened is still on origin (remote-tracking ref gone = merged+deleted or closed), rather than dropping --push per-cycle entirely or routing the record through the fixer's own PR
## Chose
114: defer the self-PR record push until no branch this loop opened is still on origin (remote-tracking ref gone = merged+deleted or closed), rather than dropping --push per-cycle entirely or routing the record through the fixer's own PR
## Rejected
Never pushing per-cycle at all (batch strictly at sprint end/halt); or piggybacking the .dacli record onto each fixer's own PR branch
## Because
A strict 'never push mid-run' policy still needs a real trigger to eventually push (the loop can run for hours), so it collapses to the same problem restated -- when is it actually safe. Checking whether the branch this cycle just opened a PR for still exists on origin (after a pruning fetch) is a precise, direct proxy for 'has GitHub's auto-merge -- which always passes --delete-branch, both in cmdPR lifecycle.go:213 and prIntegrateTask lifecycle.go:758/794 -- actually landed', reusing the exact remote-tracking-ref pattern trunkMarker/branchExists already use in this file, so it needed no new dependency (no gh calls) and no new persisted state. Piggybacking the record onto the fixer's PR was rejected because the record commit is orchestration-owned bookkeeping (task accept/close state) unrelated to any one task's diff, and folding it into a fixer's branch would make ship's belt-and-suspenders 'only .dacli staged' check (ship.go:217-225) meaningless (a fixer branch legitimately touches code too), plus it would tie the record's cadence to whichever fixer happens to be building, not to the loop's own cycle boundary.
