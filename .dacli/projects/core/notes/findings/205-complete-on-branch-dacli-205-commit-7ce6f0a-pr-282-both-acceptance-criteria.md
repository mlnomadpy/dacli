---
id: f-205-complete-on-branch-dacli-205-commit-7ce6f0a-pr-282-both-acceptance-criteria
kind: note
note_kind: finding
created: 2026-08-04T00:38:19Z
created_by: a-fixer-x41yjq
about: "[[205]]"
severity: moderate
---
# 205 complete on branch dacli/205-... commit 7ce6f0a, PR #282, both acceptance criteria met
Committed 7ce6f0a by a-fixer-x41yjq (fixer), PR #282 (auto-merge queued). Staged only the 5 intended ghmirror files (2 source + 3 test). ACCEPTANCE: (1) 'list calls detect a hit limit' — new fetchAllIssues helper (ghmirror.go) reports truncated=true when a gh issue list fetch returns exactly ghIssueListLimit(1000) records; listIssues (feeds cmdPull) now errors on truncation instead of silently returning a partial page; markerIndex.find (feeds cmdPush's marker-dedup) sets idx.truncated and cmdPush/mirrorFindingsOnly print a stdout warning after the push completes. Tests: TestListIssuesDetectsHitLimit/BelowLimitSucceeds, TestMarkerIndexDetectsHitLimit/BelowLimitIsNotTruncated (failopen_test.go), all red before the fix, green after. (2) 'synced counts only increment on verified writes' — setItemFields (project.go) previously discarded every item-edit error via '_, _ ='; it now returns bool ok (false if any attempted write errored) and cmdProject's board loop only increments synced when ok is true, for both the task and finding item loops. Test: TestSetItemFieldsReportsFailedWrites, red before/green after. go build ./... clean, go vet clean, gofmt -l . empty, go test ./... green (one pre-existing unrelated failure in internal/features/catalog traced to DACLI_AGENT env leaking into the test process from this very agent session — passes clean with env -u DACLI_AGENT, confirmed not caused by this diff). Owner: verify and close via dacli task check/done + merge PR #282.
