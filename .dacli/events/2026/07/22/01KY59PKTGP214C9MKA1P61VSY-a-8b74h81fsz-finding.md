---
id: 01KY59PKTGP214C9MKA1P61VSY
kind: event
event_kind: finding
created: 2026-07-22T16:14:22Z
created_by: a-8b74h81fsz
about: [[t-01KY59FNENE0C7CRCSXM3WH9DD]]
origin: agent
applied: true
---
ship passes bare per-project seqs as --tasks refs from a cross-project done list; multi-project workspaces resolve them ambiguously

ship.go:110 resolves the done set via store.ListTasks(w, f.Get('project'), StatusDone) — when --project is absent this spans ALL projects. doneRefs (ship.go:255-261) then emits each task's BARE t.Seq. Seqs are per-project, so two projects can both have seq N. integrate -> integrationTasks -> store.FindTask (lifecycle.go:299) matches a bare seq by strconv.Itoa(t.Seq)==ref across ALL projects (store.go:524, FindTask searches every project), so resolveRef (store.go:535-547) returns 'ref "N" is ambiguous' and integrate hard-errors. Result: in any multi-project workspace, a no-project 'dacli ship' cannot integrate — it aborts at the first colliding seq. Fix: emit project-qualified refs (e.g. NNN-slug or ULID) from doneRefs, or require/propagate --project through the whole pipeline.
