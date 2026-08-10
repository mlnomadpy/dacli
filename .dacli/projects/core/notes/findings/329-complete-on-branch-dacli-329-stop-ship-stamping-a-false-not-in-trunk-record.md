---
id: f-329-complete-on-branch-dacli-329-stop-ship-stamping-a-false-not-in-trunk-record
kind: note
note_kind: finding
created: 2026-08-10T17:13:00Z
created_by: a-fixer-g7xspa
about: "[[329]]"
severity: major
---
# 329 complete on branch dacli/329-stop-ship-stamping-a-false-not-in-trunk-record-on-every-task-it-lands
Commit b55b591 (a-fixer-g7xspa). All 4 acceptance criteria met.

Root cause: ship runs `accept --all --force` BEFORE `integrate` (forced — integrate
refuses a non-done task). acceptAll's landing check (acceptance.go, was line 267)
therefore always saw the branch as not-yet-in-trunk and durably logged "exists but
is NOT in trunk — closed anyway" on every task ship went on to merge seconds later.
--allow-unlanded only ever silenced the stderr warning, never that Log line.

Fix, three parts:

1. internal/store/landing.go (new): moved checkLanded/landingEvidence/trunkBranch
   out of acceptance into the entity layer as store.CheckLanded/LandingEvidence/
   TrunkBranch, plus two new primitives — store.ResolveBranchRef (resolves a task's
   branch to a commit sha) and store.LandingOfRef (ancestry check against an
   already-resolved sha, not a live branch name). Needed because a clean local
   merge (vcs.mergeTask) DELETES the branch once it lands — checking the branch
   name again after integrate would misread a landed task as branchless.
   acceptance/landed.go now just aliases/delegates to store (no behavior change,
   existing acceptance tests pass unmodified).

2. accept gets a new `--defer-landing` flag (acceptance.go): skips the landing
   check AND its Log line entirely (not just the warning) — refuses to combine
   with --require-verify (the two are opposite intents).

3. ship.go: accept step now passes --defer-landing. Right after computing the
   wave (before invoking integrate), ship captures each task's branch ref via
   store.ResolveBranchRef (captureWaveRefs). After integrate — on success, on a
   blocked conflict, on a hard error, or when skipped/empty — ship calls
   recordWaveLanding, which re-reads each task fresh from disk and appends the
   TRUE landingEvidence line using the captured sha's current ancestry. A
   genuinely unlanded task (conflict/error) still reads as unlanded; a landed one
   now reads "is merged into trunk", never "closed anyway".

Acceptance criteria:
(1) verified by internal/cli/TestShipDoesNotStampAFalseUnlandedRecord — real
    binary, real worker spawn+worktree+claim+done, real `dacli ship`; asserts
    the final `task show` output contains no "NOT in trunk" and does contain
    "is merged into trunk".
(2) verified by internal/cli/TestShipRecordsUnlandedTruthfullyOnConflict — a
    real merge conflict blocks the task; ship exits non-zero; the task record
    still says "NOT in trunk", never claims merged.
(3) the fix changes what's written at the old acceptance.go:267 (now guarded by
    `if !deferLanding`), not just the --allow-unlanded warning path.
(4) both new cli tests drive the real accept-then-integrate order through the
    actual compiled binary (ship's own unit tests mock the shelled subcommands
    and never exercised the landing check at all — that's why the new tests
    live in internal/cli, which can legally invoke commands from every slice).

Red-green verified by hand: git-stashed the fix (keeping only the new test
file), reran TestShipDoesNotStampAFalseUnlandedRecord — failed exactly as
predicted, task Log showed "exists but is NOT in trunk — closed anyway" on a
task the run had just merged into main. Restored the fix, went green.

PROOF: go build ./... clean, go vet ./... clean, gofmt -l . clean. go test ./...
(all packages) green under `-exec 'env -u DACLI_AGENT'`.

Owner: dacli accept 329 (task check is gated to root). PR-first is off — branch
dacli/329-stop-ship-stamping-a-false-not-in-trunk-record-on-every-task-it-lands
is ready for accept + integrate/merge --task 329.
