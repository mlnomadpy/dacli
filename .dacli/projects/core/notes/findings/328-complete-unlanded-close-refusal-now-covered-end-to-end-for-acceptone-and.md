---
id: f-328-complete-unlanded-close-refusal-now-covered-end-to-end-for-acceptone-and
kind: note
note_kind: finding
created: 2026-08-10T17:27:23Z
created_by: a-fixer-gk5xdc
about: "[[328]]"
severity: minor
---
# 328 complete: unlanded-close refusal now covered end to end for acceptOne and acceptAll
Branch dacli/328-cover-the-unlanded-close-refusal-end-to-end-acceptone-and-acceptall-under, commit 67fb84d (a-fixer-9mwg9y). All 4 acceptance criteria met by a single new file, internal/features/acceptance/unlanded_refusal_test.go — no production code changed (this was a pure coverage gap, no behavior bug).

The file was already present untracked in the worktree when I claimed the task; I verified it end to end rather than rewriting it:

(1)/(2) TestAcceptOneRefusesUnlandedUnderRequireVerify and TestAcceptAllRefusesUnlandedUnderRequireVerify: unlandedTaskFixture (landedFixture plus a branch carrying a commit never merged to trunk, mirroring issue #382's shape) drives acceptOne/acceptAll with requireVerify=true, allowUnlanded=false. Both assert clikit.ExitCode(err)==3 and the task status stays open (acceptOne) / re-reads the task from disk and asserts it is not StatusDone (acceptAll, exercising the real close path acceptance.go:223 that ship and the loop use).

(3) TestAcceptOneAllowUnlandedStillCloses / TestAcceptAllAllowUnlandedStillCloses: same fixture, same requireVerify=true, but allowUnlanded=true — both close successfully, proving the refusal is the --allow-unlanded flag's doing and not a blanket refusal on any unlanded branch.

(4) Verified by hand, not asserted in the suite (can't self-mutate production code as an acceptance criterion): temporarily changed both `if requireVerify {` occurrences (acceptance.go:167 acceptOne, :288 acceptAll — moved from the task's stated :148/:260-266 by prior commits b55b591/78eecbf, not stale on the underlying logic) to `if false {`. Reran the 4 tests: exactly the two Refuses tests failed (exit 3 never returned, tasks incorrectly closed), the two AllowUnlanded tests stayed green. Reverted with `git checkout --`, confirmed clean diff and go build green before committing.

PROOF: go build ./... clean, go vet ./... clean, gofmt -l . clean (test file was already gofmt'd). go test -exec 'env -u DACLI_AGENT' ./... green across all 40 packages.

Owner: dacli accept 328 (task check is gated to a-root; task done proposed as event since I'm not the owner). PR-first is off — branch is ready for accept + integrate/merge --task 328.
