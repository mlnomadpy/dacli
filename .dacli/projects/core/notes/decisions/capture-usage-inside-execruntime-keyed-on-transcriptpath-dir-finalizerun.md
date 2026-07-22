---
id: d-capture-usage-inside-execruntime-keyed-on-transcriptpath-dir-finalizerun
kind: note
note_kind: decision
created: 2026-07-22T16:05:16Z
created_by: a-dk00cd6m97
about: [[040]]
---
# Capture usage inside execRuntime keyed on transcriptPath dir; finalizeRun harvests detached runs
## Chose
Capture usage inside execRuntime keyed on transcriptPath dir; finalizeRun harvests detached runs
## Rejected
Change execRuntime's signature to return usage and update all call sites
## Because
verify.go calls execRuntime but is OUT of scope; writing usage.txt into filepath.Dir(transcriptPath) keeps the signature stable so spawn+supervise+verify all capture usage with zero out-of-scope edits. Detached runs can't parse live (parent returns after Release), so finalizeRun (dacli wait) self-detects a stream-json transcript and harvests its final usage — a plain-text transcript yields no result event and writes nothing, so text runtimes stay byte-for-byte unaffected.
