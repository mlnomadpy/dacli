---
id: f-sweep-result-strictness-flag-guards-which-side-the-tests-are-on
kind: note
note_kind: finding
created: 2026-08-11T19:37:54Z
created_by: a-root
severity: minor
scope: workspace
---
# Sweep result: strictness-flag guards, which side the tests are on
Applying the check from the #443 finding across internal/: for every guard gated on a strictness flag, which side of the flag do the tests exercise?

Method: grep for 'if require*', '&& require*', 'if !allow*', '&& !allow*', then for each hit disable the guard and count failures.

Results:
- acceptance unlanded close (!allowUnlanded) - WAS the defect. All tests passed requireVerify=true, so the default path had zero coverage. Fixed and covered.
- acceptance + planning empty-acceptance (!allowUnverified) - refuse-by-default and genuinely covered: disabling it turns 4 tests red.
- acceptance --require-verify / --require-independent - opt-in strictness by design. The default side is 'no guard', so there is nothing to leave untested.
- ghmirror plannedNoteCreates(requireText) - not a guard at all. It is a dry-run counting helper mirroring the real path's condition so the plan matches the action.

So the shape occurred once. Nothing else in internal/ matches, and the guards that refuse by default are all mutation-verified.

Keep the check itself: a test file where every call passes the same boolean literal for a strictness flag means the OTHER side is untested by construction - and the untested side is usually the default, which is what production runs.
