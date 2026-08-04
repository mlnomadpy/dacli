---
id: f-248-complete-on-branch-dacli-248-lesson-matcher-now-requires-2-shared
kind: note
note_kind: finding
created: 2026-08-04T10:17:42Z
created_by: a-maintainer-07d86s
about: "[[248]]"
severity: moderate
---
# 248 complete on branch dacli/248-...: lesson matcher now requires >=2 shared significant words
Fix in internal/features/insight/insight.go: lessonMatchesTask (insight.go:334) previously did strings.Contains(l.Title+l.Body lowercased, w) and returned true on the FIRST shared task word, so a single common word painted every paragraph-length lesson onto every task and a task word 'port' matched 'report'. Now it intersects the task's significantWords set with the lesson's significantWords SET (exact words, not substring) and requires minLessonOverlap=2 (new const, insight.go). ACCEPTANCE: (1) a lesson attaches only on meaningful overlap -- yes, >=2 distinct significant words; (2) match rate well below 100% -- measured on the real workspace (11 lessons x 250 open tasks = 2750 pairs) via a throwaway test: OLD 29.9% (822 matches) -> NEW 3.8% (105 matches). PROOF: added regression TestLessonMatchesTaskNeedsRealOverlap (insight_test.go) that FAILS before the change (single-word + substring cases matched) and PASSES after -- verified by git stash of insight.go. go build ./... clean; go test ./internal/features/insight/ green; go vet clean; gofmt -l internal/ empty. Full go test ./... green EXCEPT the pre-existing DACLI_AGENT env-leak in internal/features/catalog (passes under go test -exec 'env -u DACLI_AGENT'), unrelated to this change. Committed c9bed09. Owner: accept 248 to check boxes + integrate --tasks 248.
