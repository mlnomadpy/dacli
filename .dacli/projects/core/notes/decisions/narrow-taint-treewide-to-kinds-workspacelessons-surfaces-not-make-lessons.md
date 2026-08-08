---
id: d-narrow-taint-treewide-to-kinds-workspacelessons-surfaces-not-make-lessons
kind: note
note_kind: decision
created: 2026-07-22T16:25:58Z
created_by: a-n2q5ysnx5y
about: [[048]]
---
# Narrow taint TreeWide to kinds WorkspaceLessons surfaces, not make lessons surface metrics
## Chose
Narrow taint TreeWide to kinds WorkspaceLessons surfaces, not make lessons surface metrics
## Rejected
add NoteMetric to lessonKinds so a workspace metric becomes a cross-project lesson
## Because
The two files disagreed: taint marked any scope:workspace note TreeWide but WorkspaceLessons only surfaces {Decision,Finding,Ref}. Making lessons surface metrics would inject metric notes into every brief — a broad behavior change; narrowing taint via a shared SurfacesAsLesson set keeps brief output unchanged and makes the blast radius honest (a workspace metric reaches zero briefs, so it is not TreeWide).
