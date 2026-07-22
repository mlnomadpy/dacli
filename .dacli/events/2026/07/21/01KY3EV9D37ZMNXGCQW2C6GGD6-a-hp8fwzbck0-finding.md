---
id: 01KY3EV9D37ZMNXGCQW2C6GGD6
kind: event
event_kind: finding
created: 2026-07-21T23:05:49Z
created_by: a-hp8fwzbck0
about: [[t-01KY3EKR1MSTD09QSJGSW6RSTM]]
origin: agent
applied: true
---
replay reads run metadata (2 file opens) for every run dir even in single-prefix mode; FindTask hoistable out of the loop

features/execution/replay.go:82-95 — the loop over run dirs calls readRunMeta (opens invocation.txt via scanner + reads outcome.md) for EVERY run, then in the id-prefix branch discards the result for all non-matching names. Gate the file I/O behind the cheap strings.HasPrefix(e.Name(), f.Pos[0]) check so a single-run replay doesn't read metadata for the whole runs dir. Separately, replay.go:91 calls store.FindTask(w, taskRef) inside the loop although taskRef is loop-invariant — resolve it once before the loop and compare m.taskID to the cached t.ID (see the FindTask O(events×tasks) finding).
