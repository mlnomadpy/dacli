---
id: 01KY59NRTKMXVZJ1QR1JYJB5Z2
kind: event
event_kind: finding
created: 2026-07-22T16:13:54Z
created_by: a-gkrbmb023t
about: [[t-01KY59FNDY9ECT40SDBF71VWBH]]
origin: agent
applied: true
---
GradeFinding grades a non-deterministic note when two findings share a title; logSpan inflates the span across re-claim/re-complete cycles

Two quality issues. (1) store.go:810 GradeFinding matches a note by id OR by level-1 title==ref and returns the FIRST hit in os.ReadDir order (store.go:816,833). When two finding notes in a project share the same title (normal: 'two agents finding the same thing', per CreateNote's own collision comment store.go:791), verify grades whichever the FS happens to list first — a nondeterministic, possibly-wrong note gets the trust: stamp while its twin stays ungraded. Prefer id-match, and when only title matches and it is ambiguous, either grade all matches or refuse rather than silently pick one. (2) calibration.go:220 logSpan takes the FIRST 'claimed by' and the LAST 'completed by'. A task that was completed, reopened, re-claimed and re-completed yields a span from the original claim to the final completion — inflating the wall-clock actual across the idle gap. Consider the last claim before the final completion, or the first claim/first-completion pair.
