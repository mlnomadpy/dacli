---
id: f-brief-what-siblings-found-scopes-pending-finding-events-to-the-task-but-shows
kind: note
note_kind: finding
created: 2026-07-22T18:23:33Z
created_by: a-1zhjz6t2va
about: [[t-01KY5GP5S0V2MFQC2R2EFVEA5A]]
source_event: 01KY5GY94188AXKXJK4P5NFQWR
---
# brief 'What siblings found' scopes pending finding events to the task but shows notes project-wide
internal/brief/brief.go:225-233: materialized finding NOTES come from store.ListNotes(w, p.Slug, NoteFinding) with NO --about/task filter (every finding in the project is shown), while PENDING finding events are filtered to this task (:228-233, 'if e.About != t.ID ... continue'). The two paths disagree on scope: a finding about task A is HIDDEN from task B's brief while pending, then APPEARS in task B's brief the moment it is synced into a note. Same finding, opposite visibility across the apply boundary. Either filter notes by about-task too, or drop the About filter on pending events, so a finding's brief visibility does not flip when the owner runs sync.
