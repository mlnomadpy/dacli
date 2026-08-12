---
id: f-github-push-swallowed-closure-and-finding-stage-failures-before-task-394
kind: note
note_kind: finding
created: 2026-08-12T18:26:56Z
created_by: a-codex-maintainer-1weed1
about: "[[394]]"
severity: major
---
# GitHub push swallowed closure and finding-stage failures before task 394
internal/features/ghmirror/ghmirror.go previously made mirrorFindings return only a count, discarded comment read/post errors, and ignored issue-close errors; the task summary printed before mirrorDecisions, so a partial remote apply could look complete. TestPushInterruptionReportsIncompleteStagesAndRecoversIdempotently reproduces the long-window sequence and initially failed because only the later decision error was named and recovery omitted a final applied summary.
