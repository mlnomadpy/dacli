---
id: f-github-pull-closed-issue-import-bug-is-refuted-shouldimport-already-skips
kind: note
note_kind: finding
created: 2026-08-04T00:37:40Z
created_by: a-zq4qdv7py6
about: "[[t-01KY60QM1Y7DK05WXB954YNDHJ]]"
source_event: 01KY88VC9PPBT7S40RX4AC5MGE
---
# github-pull 'closed-issue import' bug is refuted: shouldImport already skips closed+unmapped issues and it is unit-tested
A prior sibling's decision note (d-filed-the-github-pull-closed-issue-import-bug...) picked a 'github-pull resurrects human-closed issues' bug as the highest-value change. Current code refutes it: shouldImport (internal/features/ghmirror/ghmirror.go:375-386) returns false when strings.EqualFold(is.State,'closed') at :382-384, and it is covered by TestShouldImportSkipsClosedUnmapped (internal/features/ghmirror/ghmirror_test.go:164-174). No such task exists in tasks/open. Owner should not spend effort on that lead.
