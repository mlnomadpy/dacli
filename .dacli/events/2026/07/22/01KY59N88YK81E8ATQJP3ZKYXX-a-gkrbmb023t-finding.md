---
id: 01KY59N88YK81E8ATQJP3ZKYXX
kind: event
event_kind: finding
created: 2026-07-22T16:13:37Z
created_by: a-gkrbmb023t
about: [[t-01KY59FNDY9ECT40SDBF71VWBH]]
origin: agent
applied: true
---
taint over-reports blast radius: a workspace-scoped METRIC note is marked TreeWide but WorkspaceLessons never surfaces metric notes

Inconsistency between two store files about which note kinds go tree-wide. taint.go:90 iterates {Finding, Decision, Ref, Metric} and taint.go:107-108 sets res.TreeWide=true for ANY scope:workspace note kind — including Metric — claiming it 'reaches every project's briefs'. But lessons.go:37 WorkspaceLessons (the code that actually surfaces workspace notes cross-project into briefs, per brief.go) iterates only {Decision, Finding, Ref} and EXCLUDES model.NoteMetric. So a scope:workspace metric note reaches ZERO other briefs yet Taint reports the whole tree as its blast radius — a false-positive over-report. Either add NoteMetric to lessons.go:37 (if a workspace metric SHOULD be a lesson) or drop the metric-triggers-TreeWide path in taint.go so the two agree.
