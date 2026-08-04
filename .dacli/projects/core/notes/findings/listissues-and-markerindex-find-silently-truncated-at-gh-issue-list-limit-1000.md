---
id: f-listissues-and-markerindex-find-silently-truncated-at-gh-issue-list-limit-1000
kind: note
note_kind: finding
created: 2026-08-04T00:37:34Z
created_by: a-fixer-x41yjq
about: "[[205]]"
severity: moderate
---
# listIssues and markerIndex.find silently truncated at gh issue list --limit 1000
internal/features/ghmirror/ghmirror.go:402 (pull's listIssues) and :1277 (push's markerIndex.find, the marker-idempotency index) both fetched gh issue list --limit 1000 and returned whatever came back with no check — a repo with >=1000 issues silently got only the first page: pull adopted nothing past it, and push's dedup index missed old issues past the page, risking duplicate issue creation on retry. Fixed by a shared fetchAllIssues helper that reports len(issues)>=1000 as truncated; listIssues now errors (pull refuses a partial adoption) and markerIndex now warns on stdout after a push (writes already made are fine, but adoption past the page was unchecked).
