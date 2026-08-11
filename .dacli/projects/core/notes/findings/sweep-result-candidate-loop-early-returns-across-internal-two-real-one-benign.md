---
id: f-sweep-result-candidate-loop-early-returns-across-internal-two-real-one-benign
kind: note
note_kind: finding
created: 2026-08-11T16:26:14Z
created_by: a-root
about: "[[[[363]]]]"
severity: minor
scope: workspace
---
# Sweep result: candidate-loop early returns across internal/ — two real, one benign, none else
Method: every 'for ... range' in internal/ (non-test), body walked with brace-depth tracking, flagging a return of a negative or default value at LOOP-BODY indentation — i.e. unconditional for that candidate, so no later candidate can execute. Indentation depth is what distinguishes the shape from the correct 'return inside an if' form, which a plain grep cannot.

Results across the whole tree:

- internal/store/landing.go LandingOfRef — REAL, fixed. Returned LandingUnlanded from inside the loop, making refs/heads/<trunk> unreachable whenever origin/<trunk> existed.
- internal/store/landing.go ResolveBranchRef — REAL, fixed. Returned the first EXISTING ref, certifying unmerged deliverables as landed.
- internal/features/execution/execution.go:1814 (cmdRunsShow) — BENIGN. The 'return nil' is a success after printing the first run whose id carries the given prefix; first-match-wins is the intended semantics for a prefix lookup and no later candidate is owed a turn.
- internal/store/similarity.go:146 and internal/workspace/workspace.go:230 — BENIGN. Both return only on error, which is the correct early exit.

So the shape occurred twice, both in landing.go, both now fixed and tested. Nothing else in internal/ matches.

One adjacent inconsistency noticed and NOT fixed: cmdRunsShow takes the first prefix match silently, while FindTask refuses an ambiguous ref outright. A run-id prefix matching two runs shows one without saying so. Minor, and a different fault class from this sweep — recorded here so it is not lost.
