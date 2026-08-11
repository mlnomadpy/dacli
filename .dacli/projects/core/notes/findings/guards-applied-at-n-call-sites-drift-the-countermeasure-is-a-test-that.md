---
id: f-guards-applied-at-n-call-sites-drift-the-countermeasure-is-a-test-that
kind: note
note_kind: finding
created: 2026-08-11T16:20:07Z
created_by: a-root
severity: major
scope: workspace
origin: internal/store/containment_test.go
---
# Guards applied at N call sites drift; the countermeasure is a test that enumerates the sites
CreateShortcut and CreateRuntime took a caller-supplied name straight into a filename with no SafeSegment guard, while CreateQueue, CreateRole and CreateProject all carried one. The shortcut case was asymmetric on top: RemoveShortcut refuses a traversing name, so a file created that way could not be deleted by the tool that wrote it.

This is the fourth instance of the same failure mode in this repo (Flags.Reject reached 4 handlers of 112; four verified grant bypasses shipped next to correctly-gated siblings). Fixing the two call sites would leave the fifth to be found the same way.

The countermeasure that works here is arch_test.go's: enumerate the call sites in a TEST, so adding a new constructor without the guard fails CI rather than waiting to be noticed. containment_test.go drives every name-taking constructor with five traversal shapes and asserts both the refusal AND that no file appeared outside .dacli - measuring containment rather than trusting error text.

It also pins the containment CreateRisk/CreateNote/CreateTask get INDIRECTLY (they require the project to load, so a traversing project resolves to no project.md), because a guard that is a side effect of a lookup would open silently if the lookup ever became optional.
