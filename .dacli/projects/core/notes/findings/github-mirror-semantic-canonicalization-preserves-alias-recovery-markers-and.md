---
id: f-github-mirror-semantic-canonicalization-preserves-alias-recovery-markers-and
kind: note
note_kind: finding
created: 2026-08-13T22:27:29Z
created_by: a-fixer-btcg7r
about: "[[437]]"
severity: major
---
# GitHub mirror semantic canonicalization preserves alias recovery markers and merged evidence
internal/features/ghmirror/ghmirror.go canonicalNoteFiles groups normalized near-duplicate titles before plannedNoteCreates and both decision/finding write stages; findNoteMarker searches every grouped source id, and noteFileText retains distinct evidence. Focused mutation failed with undefined canonicalNoteFiles before implementation; go test ./internal/features/ghmirror and go test ./... pass.
