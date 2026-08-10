---
id: d-filed-the-untested-unlanded-refusal-composition-in-acceptance-go-as-the
kind: note
note_kind: decision
created: 2026-08-10T16:40:32Z
created_by: a-mutation-auditor-dgweq2
about: "[[324]]"
---
# Filed the untested unlanded-refusal composition in acceptance.go as the surviving mutation; reported the governor guards and spawn refusal suites as mutation-resistant
## Chose
Filed the untested unlanded-refusal composition in acceptance.go as the surviving mutation; reported the governor guards and spawn refusal suites as mutation-resistant
## Rejected
Filing a mutation against orchestration/guards_test.go or execution/spawn_refusal_test.go, or filing generic 'add coverage' work
## Because
guards_test.go and spawn_refusal_test.go RESISTED every mutation I could construct: each spawn-refusal case asserts a SPECIFIC message substring ('WIP limit (1/1)', 'above role junior's cap', 'cannot enforce read-only') plus zero-side-effect counts (countAgents/countRuns), so inverting any one guard changes the message another case pins or leaks a side effect; the governor suite asserts exact post-state (ZeroStreak, Cycle, WindowSpent, errCorruptState identity) and even documents why it avoids a weakening conjunction (guards_test.go:365-371). The acceptance suite is the outlier: checkLanded (the detector) and unlandedRefusal (the message) are each unit-tested in isolation, but the branch composing them into an actual close refusal (acceptance.go:147-150, mirrored 261-264) is driven by no test, so a one-line 'if requireVerify {'->'if false {' silently degrades the exit-3 refusal to a warn-and-close and the whole suite stays green — a real, named, verifiable hole rather than speculative coverage work, and it guards the highest-cost property in the slice (recording unlanded work as done under the operator's strict flag, the issue #382 class).
