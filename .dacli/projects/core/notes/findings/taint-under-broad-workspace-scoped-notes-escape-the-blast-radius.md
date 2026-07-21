---
id: f-taint-under-broad-workspace-scoped-notes-escape-the-blast-radius
kind: note
note_kind: finding
created: 2026-07-21T21:31:33Z
created_by: a-root
about: [[006]]
severity: major
---
# taint under-broad: workspace-scoped notes escape the blast radius
REVIEWER a-0m1fb0yvgh (opus, via dacli spawn): a --scope workspace poisoned note reaches EVERY project's brief as a 'Lesson from other projects' (brief.go:171, lessons.go:37) but Taint marks only the note's home project (taint.go:87), and only NoteFinding sets res.Projects — a workspace-scoped tainted decision/ref taints zero projects in the report.
