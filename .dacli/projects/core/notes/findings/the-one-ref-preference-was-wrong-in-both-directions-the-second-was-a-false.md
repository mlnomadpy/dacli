---
id: f-the-one-ref-preference-was-wrong-in-both-directions-the-second-was-a-false
kind: note
note_kind: finding
created: 2026-08-11T16:25:03Z
created_by: a-root
about: "[[[[363]]]]"
severity: major
scope: workspace
origin: internal/store/landing.go
---
# The one-ref preference was wrong in BOTH directions; the second was a false landed
Sweeping for the shape of the LandingOfRef bug found the same fault one level up, in ResolveBranchRef, with a worse consequence.

ResolveBranchRef returned the first branch ref that EXISTED (origin/<branch>). A branch pushed once and then advanced locally has its deliverable only in refs/heads/<branch>. If the pushed part had already merged - the ordinary case, since it is what got reviewed - the stale origin sha genuinely IS an ancestor of trunk, so CheckLanded reported LandingLanded and accept closed the task recording that the work was in trunk. It was not.

A false unlanded is noise. A false landed is the record certifying something untrue, and it is the exact failure issue #382 exists for.

Two things worth carrying forward:

1. The sweep paid for itself immediately. Having named the shape (a candidate loop that concludes from the first candidate that RESPONDS rather than the first that ANSWERS), the second instance was found by looking twenty lines up from the first, not by another audit cycle. When a defect is fixed, grep the shape before closing the task.

2. There was no ordering of the two refs that fixed it - the --pr path merges what origin has, the local-merge path merges what refs/heads has. When two sources disagree and neither is authoritative, the fix is to stop choosing: consult all of them and take the conservative combination (Unknown > Unlanded > Landed). Reordering would have moved the bug, not removed it.
