---
id: f-205-complete-on-branch-pagination-hit-limit-verified-sync-counts-across-both
kind: note
note_kind: finding
created: 2026-08-04T10:18:34Z
created_by: a-maintainer-56k75j
about: "[[205]]"
severity: moderate
---
# 205 complete on branch — pagination hit-limit + verified sync counts across BOTH the issue and board surfaces
Branch dacli/205-fix-remaining-ghmirror-audit-findings-pagination-truncation-and-unverified-sync. Two commits: 7ce6f0a (prior fixer: issue-list surface) + 146e049 (mine: board-sync surface). ACCEPTANCE both satisfied and tested. (1) list calls detect a hit limit: gh issue list via fetchAllIssues/ghIssueListLimit — listIssues (ghmirror.go:435) ERRORS on a hit-limit fetch for pull, markerIndex.warnIfTruncated (ghmirror.go:1319) warns on push; AND the board item-list via itemSnapshot/projectItemListLimit (project.go:251) REFUSES a truncated snapshot before ensureItem can item-add duplicates (project.go:411). (2) synced counts only increment on verified writes: setItemFields (project.go:593) now returns bool and cmdProject only does synced++ when the write landed (project.go:441,466). PROOF: go build ./... clean; go test -exec 'env -u DACLI_AGENT' ./... all green (the sole non-stripped failure, catalog's TestCatalogRefuses..., is the known DACLI_AGENT test-isolation leak, not this change); go vet clean; gofmt -l internal/ empty. Bug reproduced: TestItemSnapshotRefusesOnHitLimit FAILS with the guard neutralized, PASSES with it. Owner: dacli accept 205 then integrate/merge --task 205.
