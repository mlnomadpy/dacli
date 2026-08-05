---
id: f-289-complete-on-branch-dacli-289-empty-acceptance-close-refused-on-all-three
kind: note
note_kind: finding
created: 2026-08-04T20:42:58Z
created_by: a-maintainer-wyr06h
about: "[[289]]"
severity: moderate
---
# 289 complete on branch dacli/289-...: empty-acceptance close refused on all three paths, commit 0b434a2
Fix for the zero-boxes-counts-as-all-boxes close. Root cause: task done's unmet-box scan (planning.go) finds an empty list on a criteria-less task and passes; accept's CheckAllAcceptance checks 0 boxes and reports success; the propose->sync route (sync.go EventProposeStatus) MoveTask's to done with no acceptance check at all. Fix: added shared read-only predicate store.HasAcceptanceCriteria (store.go:283, len(t.Acceptance())>0), consulted at each close path so the rule is identical on all three: (1) task done (planning.go cmdTaskDone) refuses exit 3 before the propose branch unless --allow-unverified; (2) accept (acceptance.go acceptOne refuses exit 3, acceptAll skips) unless --allow-unverified; (3) sync (sync.go:146 EventProposeStatus) leaves a done proposal on an empty-acceptance task PENDING like the malformed case rather than auto-closing. --allow-unverified closes but stamps the Log 'closed with NO acceptance criteria — UNVERIFIED' so the record never implies verification. Tests fail-before/pass-after: planning_test.go (refuse/allow-unverified/positive), acceptance_test.go (refuse/allow-unverified), sync_test.go (refuse-empty/apply-with-acceptance); verified reproduction by removing the planning guard -> TestTaskDoneRefusesEmptyAcceptance FAILs (task closed with zero verification), restored -> green. go build/test/vet all green, gofmt clean. PR-first off; owner: dacli accept 289 then merge --task 289. NOTE loop path reconcilePendingAccepts shells 'accept --force' so it inherits the guard.
