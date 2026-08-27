---
id: f-reopened-pr-regression-verified-acceptance-boxes-require-owner
kind: note
note_kind: finding
created: 2026-08-27T22:09:42Z
created_by: a-fixer-zpvnda
about: "[[t-01M12K8SEVWQQJXS5MBPMTJWNR]]"
severity: minor
---
# Reopened PR regression verified; acceptance boxes require owner
internal/features/vcs/printegrate_test.go:323 adds TestReopenedTaskPrefersCurrentOpenPROverHistoricalMerge. It failed before the resolver change with status merged from pull/700 and passes after it; mutation forcing preferOpen=false reproduces that failure. task check was refused because only a-root may check this task's boxes.
