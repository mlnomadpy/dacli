---
id: f-sync-go-logonce-comment-claims-a-fresh-per-pass-reload-that-no-longer-happens
kind: note
note_kind: finding
created: 2026-07-22T16:17:27Z
created_by: a-0htd7wjqyt
about: [[t-01KY59FNFAEE0KT7PWV8HAAY4A]]
source_event: 01KY59TAFKB2JEPGKDV2N701H7
---
# sync.go logOnce comment claims a fresh-per-pass reload that no longer happens
internal/eventlog/sync.go:184-185 — logOnce's comment ends 't.Doc reflects the durable Log because Sync loads the task fresh each pass.' That is stale: Sync no longer resolves each event with FindTask (a fresh disk read); it builds one store.BuildTaskIndex up front (:36) and idx.Find returns a shared *Task pointer reused across every event in the pass (store.go:617). The idempotency invariant STILL holds — each Sync invocation rebuilds the index from disk, and within a pass the shared pointer is mutated in place so t.Doc tracks applied state — but for the RIGHT reason (shared-pointer mutation + per-invocation rebuild), not the reason the comment gives (per-event reload). Because this comment sits on the load-bearing idempotency path (the thing that stops a re-run duplicating 'claimed by' lines and finding notes), the misleading rationale is a real trap for the next editor. Correct the comment to describe the index-shared-pointer model.
