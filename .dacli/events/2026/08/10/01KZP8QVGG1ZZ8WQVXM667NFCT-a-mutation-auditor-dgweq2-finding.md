---
id: 01KZP8QVGG1ZZ8WQVXM667NFCT
kind: event
event_kind: finding
created: 2026-08-10T16:40:21Z
created_by: a-mutation-auditor-dgweq2
about: "[[t-01KZP8HAMD22FJRA35NZBRK614]]"
origin: agent
applied: true
---
accept's unlanded-branch refusal is wired but wholly untested: a one-line mutation closes unlanded work green under --require-verify

SUITE AUDITED: internal/features/acceptance/{acceptance_test.go, evidence_test.go, landed_test.go} — the accept/close safety suite.

WHAT EACH TEST ACTUALLY ASSERTS (not what its name promises):
- landed_test.go TestUnlandedBranchIsDetected (56): calls checkLanded() DIRECTLY, asserts it returns landingUnlanded. Unit test of the detector only.
- landed_test.go TestUnlandedRefusalNamesTheWayOut (107): calls unlandedRefusal() DIRECTLY, asserts exit 3 + names --allow-unlanded. Unit test of the message builder only.
- landed_test.go TestMergedBranchReadsAsLanded / TestNoBranchIsNotAFailure: checkLanded() unit, landingLanded / landingNoBranch.
- evidence_test.go TestRequireVerifyRefusesUnverifiedClose (80): drives acceptOne with requireVerify=true but NO --verify cmd -> hits the requireVerifyRefusal at acceptance.go:124-126, NOT the unlanded gate. The task has no branch (landingNoBranch), so line 147 is never entered.
- acceptance_test.go / e2e_test.go accept cases: every fixture task either has no branch (landingNoBranch) or no git (landingUnknown). NONE creates a branch with commits absent from trunk and then drives acceptOne/acceptAll.

THE HOLE: the COMPOSITION at acceptance.go:146-158 (acceptOne) and 260-266 (acceptAll) — 'landing==landingUnlanded && !allowUnlanded { if requireVerify { return unlandedRefusal(...) } }' — is exercised by ZERO tests. checkLanded and unlandedRefusal are each tested alone; the branch that turns a detected unlanded state into an actual close refusal is not.

SURVIVING MUTATION (one line, acceptance.go:148): change 'if requireVerify {' -> 'if false {' (equivalently delete the 'return' at :149). The unlandedRefusal at :149 becomes unreachable; an unlanded task under --require-verify degrades from an exit-3 refusal to a stderr warning + silent close. The whole acceptance suite stays GREEN: no test drives acceptOne/acceptAll with landing==landingUnlanded, TestUnlandedRefusalNamesTheWayOut still passes (calls unlandedRefusal directly), TestUnlandedBranchIsDetected still passes (calls checkLanded directly). Parallel unprotected site: acceptAll acceptance.go:261-264. An even broader single-line survivor: flip :147 to 'if landing == landingLanded && ...' — kills both the refusal AND the warning for unlanded tasks, still green (landingEvidence at :176 is computed from 'landing' independently, so it still compiles and the log line is unaffected).

UNPROTECTED BEHAVIOUR: 'an unlanded branch under --require-verify must REFUSE the close (exit 3)'. This gate exists specifically to prevent issue #382's failure (done:15/21 reported while the commands did not exist because the PRs failed to merge; see the docstring at landed.go:26-36).

CONCRETE USER FAILURE: an operator runs 'dacli accept 042 --verify "go build && go test" --require-verify' on a task whose PR failed to merge (branch has commits not in trunk). With the mutation live, go build/test pass (the TREE compiles), the unlanded gate silently degrades to a warning, and the task is marked done with 'completed by' actuals — recording work the trunk never received while the operator's strictest flag reported success. Exactly the record-disagreeing-with-reality class --require-verify is meant to make impossible.

FIX BAR (for the strengthened test the implementer must add): a new integration test, e.g. TestAcceptRefusesUnlandedBranchUnderRequireVerify, must build the landed_test.go landedFixture, create the task branch with a commit NOT merged to main (as TestUnlandedBranchIsDetected already does), then call acceptOne(ctx,w,root,task,"true",/*requireVerify*/true,false,false,/*allowUnlanded*/false) and assert: clikit.ExitCode(err)==3, err mentions --allow-unlanded (or 'NOT in main'), and the task stays open (status != done). It MUST fail against the :148 mutation ('if false {') — which returns nil + closes the task — and pass against the real code. A sibling test with allowUnlanded=true must still close (proves the flag, not a blanket refusal). Add the mirror for acceptAll at acceptance.go:260-266.
