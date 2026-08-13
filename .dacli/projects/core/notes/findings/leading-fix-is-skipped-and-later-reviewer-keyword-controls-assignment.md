---
id: f-leading-fix-is-skipped-and-later-reviewer-keyword-controls-assignment
kind: note
note_kind: finding
created: 2026-08-13T12:51:02Z
created_by: a-fixer-2hbsam
about: "[[412]]"
severity: major
---
# Leading Fix is skipped and later reviewer keyword controls assignment
internal/features/teamops/teamops.go:636 omits fix from kindVerbs while inferKind scans two title words; the new TestTeamAssignLeadingIntentVerbTakesPrecedence fails because Fix verify/audit/review titles all route to reviewer.
