---
id: 01KY88VC9PPBT7S40RX4AC5MGE
kind: event
event_kind: finding
created: 2026-07-23T19:57:12Z
created_by: a-zq4qdv7py6
about: [[t-01KY60QM1Y7DK05WXB954YNDHJ]]
origin: agent
applied: false
---
github-pull 'closed-issue import' bug is refuted: shouldImport already skips closed+unmapped issues and it is unit-tested

A prior sibling's decision note (d-filed-the-github-pull-closed-issue-import-bug...) picked a 'github-pull resurrects human-closed issues' bug as the highest-value change. Current code refutes it: shouldImport (internal/features/ghmirror/ghmirror.go:375-386) returns false when strings.EqualFold(is.State,'closed') at :382-384, and it is covered by TestShouldImportSkipsClosedUnmapped (internal/features/ghmirror/ghmirror_test.go:164-174). No such task exists in tasks/open. Owner should not spend effort on that lead.
