---
id: 01KZP42RMB5VKWJ4WFTWAQEDZK
kind: event
event_kind: finding
created: 2026-08-10T15:18:55Z
created_by: a-go-auditor-qz3zb9
about: "[[t-01KZ93DW7P6BQ1HHDWY7MEH2KJ]]"
origin: agent
applied: true
---
unknown --status silently lists zero tasks instead of refusing

store.listTasksRaw (internal/store/store.go:833-836) walks model.AllStatuses and skips any folder where 'status != "" && st != status'. An unrecognized status value (e.g. --status opne / closed / done-ish typo) equals none of the four canonical statuses (model.go:100-107, which defines no Valid() method), so EVERY folder is skipped and the function returns an empty slice with a nil error. Callers pass raw user input with zero validation: task list (planning.go:234), task list --json (planning.go:277), and lint (insight.go:106) all do model.Status(f.Get("status")). Concrete failure: 'dacli task list --status closed' (user means done) prints nothing and exits 0 — reads as 'backlog empty' when the filter was garbage. Same for 'dacli lint --status <typo>' which then reports a clean sweep having examined nothing. This is a silent-wrong-answer: the record (an empty list, exit 0) disagrees with reality (there are tasks). Fix: add model.Status.Valid()/parse and refuse an unknown value (usage/exit 2) at the flag layer, mirroring how f.Reject already refuses unknown flag NAMES; an unknown flag VALUE deserves the same loud refusal, not a plausible empty answer.
