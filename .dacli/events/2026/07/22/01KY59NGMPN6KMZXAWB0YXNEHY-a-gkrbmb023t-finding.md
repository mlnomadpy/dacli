---
id: 01KY59NGMPN6KMZXAWB0YXNEHY
kind: event
event_kind: finding
created: 2026-07-22T16:13:46Z
created_by: a-gkrbmb023t
about: [[t-01KY59FNDY9ECT40SDBF71VWBH]]
origin: agent
applied: true
---
Slugify returns empty string for punctuation-only or non-ASCII titles, producing '.md' filenames and 'f-'/'d-' ids

internal/store/store.go:33 Slugify only keeps [a-z0-9] and collapses everything else to dashes, then trims. A title with no ASCII alphanumerics (punctuation-only e.g. '???', or a non-Latin title e.g. Japanese/Arabic/CJK) yields slug=='' (store.go:60). Downstream: CreateNote (store.go:789/747) writes path filepath.Join(NotesDir, ''+'.md') = a hidden '.md' file with id prefix+'' (e.g. 'f-'); CreateTask (store.go:311) writes '001-.md'. Two such notes collide, then the collision branch (store.go:792) makes '-xxxxxx.md' with id 'f--xxxxxx'. Non-ASCII titles are realistic (i18n). Guard: if slug=='' fall back to a stable token (e.g. a short hash of the title or a ulid) so every object keeps a legible, unique filename and id.
