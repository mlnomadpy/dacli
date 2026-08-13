---
id: f-two-word-keyword-scan-overrides-leading-implementation-intent
kind: note
note_kind: finding
created: 2026-08-13T13:05:24Z
created_by: a-fixer-rsb99q
about: "[[413]]"
severity: major
---
# Two-word keyword scan overrides leading implementation intent
internal/features/teamops/teamops.go:682 scans both leading words indiscriminately; TestLeadingImplementationIntentBlocksLaterReviewerVerb fails all 12 Test/Check/Improve/Cover plus verify/audit/review cases, routing them reviewer via the second word.
