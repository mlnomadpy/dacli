---
id: 01KZ6STK6AEZA2BAFHH556NGY5
kind: event
event_kind: finding
created: 2026-08-04T16:31:05Z
created_by: a-reviewer-664htb
about: "[[t-01KZ6S9FVG533ABEVXZBZZ7SC3]]"
origin: agent
applied: true
---
close: dacli task done closes a task with an EMPTY acceptance section with zero verification; manual done/accept has no trunk-landing gate (loop-only 115 fix does not cover hand-driven agents)

Stage: CLOSE. cmdDone (planning.go:396-406) refuses only on UNCHECKED boxes: it iterates t.Acceptance() collecting unmet; if the acceptance section is EMPTY, unmet is nil, and CloseTask runs -- a task with zero acceptance criteria closes with zero verification, despite the comment 'done VERIFIES, not just records' (planning.go:396). (Task 115's own file is an instance: it closed with an empty ## Acceptance section.) Separately, neither task done (planning.go:410 CloseTask) nor accept (acceptance.go:114-157) gates on the branch having merged to trunk -- CloseTask moves to done/ on box-satisfaction alone, so a hand-driven rw owner can mark done while its branch is orphaned off main. ALREADY FILED for the loop path: task 115 (loop closed on PR-open not merge) is done, and reconcilePendingAccepts (orchestration.go:724-736) defers accept --force until prLandStatus=='merged'; tasks 154/157/159/160 (orphan re-integrations, PushSync/FastForward) are also done -- do NOT re-file those. The residual NOT covered by 115: (a) the empty-acceptance trivial close, and (b) the manual (non-loop) done/accept path still has no merge-to-trunk gate. Consider a doctor check for done tasks whose branch is not an ancestor of main.
