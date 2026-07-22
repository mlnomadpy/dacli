---
id: 01KY3ET4P4J1DAQZWC530BMG2M
kind: event
event_kind: finding
created: 2026-07-21T23:05:11Z
created_by: a-hp8fwzbck0
about: [[t-01KY3EKR1MSTD09QSJGSW6RSTM]]
origin: agent
applied: true
---
Single-item Load* helpers read the entire directory off disk (LoadRole/LoadRuntime/LoadShortcut → LoadAll*)

roles.go:113 LoadRole, runtimefiles.go:132 LoadRuntime, shortcutfiles.go:89 LoadShortcut each delegate to their LoadAll* sibling, which mdstore.ReadFile+Parses EVERY file in the directory, then linear-scans for one name — even though the exact path is already computable (w.RolePath(name), etc.). roles.go:164 ActiveInRole re-reads every agent file on each call. For a one-shot lookup it's tolerable, but any caller resolving several names in a loop (per-agent role/runtime resolution) becomes O(items × files) disk reads. Read the single named file directly.
