---
id: f-r4-regression-check-all-prior-in-scope-fixes-held-insight-brief-spm-gates
kind: note
note_kind: finding
created: 2026-07-22T18:23:33Z
created_by: a-1zhjz6t2va
about: [[t-01KY5GP5S0V2MFQC2R2EFVEA5A]]
source_event: 01KY5GYNDWJANSWTPVRY2ERQJN
---
# R4 regression check: all prior in-scope fixes held (insight/brief/spm/gates)
Verified by reading current source (build/exec sandboxed). HELD FIXED: (1) blocked-task consistency - cmdCriticalPath now excludes blocked exactly like cmdNext (insight.go:433-438) and both edge loops edge only to openIDs (insight.go:182, :453), so an open task depending on a blocked task no longer trips 'edge references unknown task'. (2) decisions gate now counts only notes with a non-empty Rejected section (gates.go:393-400), Why='M of N recorded carry a rejection' - a rejection-free decision no longer clears the design gate. (3) spm.Network.Parallelizable doc rewritten to state it does NOT filter by dependency readiness (criticalpath.go:259-268). (4) kahn uses a posHeap min-frontier, no per-pop re-sort (criticalpath.go:208-257); maskCode uses bytes.Index, no whole-tail string copy (ambiguity.go:222). (5) brief.trim keeps a running byte total instead of re-rendering per drop, sectionLen kept in lockstep with render (brief.go:417-444). (6) 'What siblings found' now honors MillerCap over notes+pending with an announced omission and trust-floor over shown findings (brief.go:236-273). (7) per-band n>=10 gate on both size and agent bands - a thin band prints median + 'provisional, n<10 - no calibrated range', killing the n=1 'x0.03-x0.03' theater (insight.go:665,:699,:733). (8) doctor broken-calibration-span + duplicate-task-file checks present (insight.go:851,:881). No regressions observed. Open new issues filed separately.
