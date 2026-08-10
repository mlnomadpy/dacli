---
id: f-328-complete-on-branch-dacli-328-cover-the-unlanded-close-refusal-end-to-end
kind: note
note_kind: finding
created: 2026-08-10T17:26:40Z
created_by: a-fixer-9mwg9y
about: "[[328]]"
severity: major
---
# 328 complete on branch dacli/328-cover-the-unlanded-close-refusal-end-to-end-acceptone-and-acceptall-under
Commit 67fb84d (a-fixer-9mwg9y). All 4 acceptance criteria met.

New file internal/features/acceptance/unlanded_refusal_test.go, 4 tests, reusing landedFixture/git helpers from landed_test.go (real git repo, real branch history — not stubs):

(1) TestAcceptOneRefusesUnlandedUnderRequireVerify: builds a task branch with a commit NOT merged to trunk (main), drives acceptOne(verify="true", requireVerify=true, allowUnlanded=false) directly, asserts err != nil, clikit.ExitCode(err)==3, and task.Status != done.

(2) TestAcceptAllRefusesUnlandedUnderRequireVerify: same fixture, files a propose event (the channel acceptAll's proposedTasks() reads) via propose(), drives acceptAll(verify="true", requireVerify=true, allowUnlanded=false) — the batch path ship/loop use (currently acceptance.go:283-294, was 260-266 in the task brief's pre-329 numbering; store.CheckLanded/AppendLog only mutate the in-memory Doc, so the early unlandedRefusal return before CloseTask leaves the on-disk task genuinely untouched/open). Asserts exit 3 and re-reads the task from disk to confirm status != done.

(3) TestAcceptOneAllowUnlandedStillCloses and (4) TestAcceptAllAllowUnlandedStillCloses: identical fixtures with allowUnlanded=true, assert err==nil and status==done — proving the refusal is the flag's doing, not a blanket refusal on every unlanded branch.

Red-green verified by hand, both requireVerify gates independently: mutated acceptOne's 'if requireVerify {' (now acceptance.go:167, was :148 pre-329) to 'if false {' alone -> TestAcceptOneRefusesUnlandedUnderRequireVerify failed exactly as predicted ('acceptOne closed a task whose branch is NOT in trunk under --require-verify'); reverted (git checkout), then mutated acceptAll's copy (acceptance.go:288) alone -> TestAcceptAllRefusesUnlandedUnderRequireVerify failed the same way; reverted again and confirmed git diff was clean before committing.

PROOF: go build ./... clean, go vet ./... clean, gofmt -l internal/features/acceptance/ clean. go test ./internal/features/acceptance/... -count=1 -v: all 19 tests green (15 pre-existing + 4 new). Full go test ./...: only pre-existing failures are in internal/features/briefing (TestCatchup*), a known ambient-DACLI_AGENT dogfood-session artifact unrelated to this change (acceptance package and every other package green).

Owner: dacli accept 328 (task check is gated to a-root; I could not check the boxes myself).
