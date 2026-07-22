---
id: d-landed-derives-from-ship-s-own-integrated-n-branch-es-stdout-line-not-a
kind: note
note_kind: decision
created: 2026-07-22T23:04:58Z
created_by: a-77q7eps4da
about: [[085]]
---
# landed derives from ship's own 'integrated N branch(es)' stdout line, not a StatusDone-count delta
## Chose
landed derives from ship's own 'integrated N branch(es)' stdout line, not a StatusDone-count delta
## Rejected
keep counting the done-task delta and just fix the docstring to say it measures acceptance-closure
## Because
the acceptance requires the thrash guard to observe real trunk merges under the default --pr --auto path, where accept --all closes a task immediately on proposal while ship queues GitHub auto-merge and merges nothing in-cycle (lifecycle.go prIntegrateTask returns landed=false, vcs/lifecycle.go:611-621); a done-count delta would count that queued state as progress forever and could never trip NoProgressHalt. ship/integrate already print the real trunk-merge count on a stable 'integrated %d branch(es)' line (vcs/lifecycle.go:543, ship.go:325 integratedCount) distinct from the 'queued %d PR(s)'/'opened %d PR(s)' lines for not-yet-landed work, so runCycle now captures ship's combined stdout and parses that line (orchestration.go landedCount) instead of diffing task status
