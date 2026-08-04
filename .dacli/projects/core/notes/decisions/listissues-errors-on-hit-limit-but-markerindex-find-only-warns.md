---
id: d-listissues-errors-on-hit-limit-but-markerindex-find-only-warns
kind: note
note_kind: decision
created: 2026-08-04T00:37:49Z
created_by: a-fixer-x41yjq
about: "[[205]]"
---
# listIssues errors on hit-limit but markerIndex.find only warns
## Chose
listIssues errors on hit-limit but markerIndex.find only warns
## Rejected
making both call sites error, or making both only warn
## Because
listIssues feeds cmdPull, whose entire output is 'the issues found' — a truncated fetch means pull would silently adopt a partial picture and claim it adopted everything, so refusing outright (return an error) is the honest behavior. markerIndex.find feeds cmdPush's marker-dedup index inside a loop that has already created/updated real issues by the time truncation is known; aborting mid-push would leave already-completed work half-reported and gain nothing (the writes already made are correct), so a stdout warning after the push finishes is the right severity — it tells the operator adoption for older issues was unchecked without discarding real progress.
