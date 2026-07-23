---
id: 01KY7VNCGB0QJJN8F0N30G0M77
kind: event
event_kind: finding
created: 2026-07-23T16:06:45Z
created_by: a-xrcxmhwz96
about: [[t-01KY7VMYD97RTRRZFVZKZJ4ZV2]]
origin: agent
applied: false
---
runtimefiles comma-split root cause is write-side missing quotes, not read-side

Confirmed round-trip defect. WRITE: internal/store/runtimefiles.go:79-83 setInline does d.Front.Set(k, '['+strings.Join(v, ', ')+']') and never quotes individual elements. READ: mdstore.splitTop (internal/mdstore/mdstore.go:176-205) already respects quotes (inQ) and only splits on commas at depth 0 outside quotes. So the read path is CORRECT; the fix belongs entirely on the write side (quote any element containing a comma). Evidence: default preset execution.go:62 SandboxRO=[--allowedTools, Read,Grep,Glob,LS,Bash(dacli:*)]; on load GetList returns 6 elements not 2. Real incident: .dacli/runtimes/cc.md:10 was HAND-corrected to add quotes (cc.md:18 'runtime add mangled --allowedTools'); any future CreateRuntime/rewrite re-mangles it. No round-trip test exists in internal/store (no runtimefiles_test.go).
