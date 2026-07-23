---
id: 01KY7ENAK9H079DRTASZTD401D
kind: event
event_kind: finding
created: 2026-07-23T12:19:31Z
created_by: a-m146x20e8d
about: [[t-01KY60QM1Y7DK05WXB954YNDHJ]]
origin: agent
applied: false
---
github pull imports human-CLOSED issues as fresh open tasks (State ignored)

internal/features/ghmirror/ghmirror.go: listIssues (line 384) fetches 'gh issue list --state all', so the result includes CLOSED issues. shouldImport (lines 371-378) only checks the mapped-set and the dacli marker; it never reads is.State. The State field is declared on ghIssue (line 355) but read nowhere in the package (grep 'State' -> only the struct decl). Result: a human-authored issue that a maintainer already CLOSED (wontfix/duplicate/resolved) is adopted by 'github pull' as a brand-new OPEN local task, resurrecting settled work; on 'github sync' the subsequent push would then re-mirror it. This is asymmetric with the push half, which closes the issue on task done. TestShouldImportSkipLogic (ghmirror_test.go:136) covers marker-skip and mapped-skip only; no test sets is.State, so closed-issue import is untested and almost certainly unintended. Fix: shouldImport should skip is.State=='closed' unless the issue is already mapped to a local task.
