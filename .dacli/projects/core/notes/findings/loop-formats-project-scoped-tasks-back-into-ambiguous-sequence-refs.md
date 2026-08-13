---
id: f-loop-formats-project-scoped-tasks-back-into-ambiguous-sequence-refs
kind: note
note_kind: finding
created: 2026-08-13T15:52:59Z
created_by: a-fixer-f6typj
about: "[[423]]"
severity: major
---
# Loop formats project-scoped tasks back into ambiguous sequence refs
internal/features/orchestration/orchestration.go:874 passes fmt.Sprintf("%03d", t.Seq) to implementation spawn and ensureImproveTask returns the same short form for review anchors. TestDriverUsesStableTaskIDsAcrossProjects fails on unfixed code: build spawn task ref = "001", want stable ID.
