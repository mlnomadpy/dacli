---
id: 01KY59SR9XT0BFNAG6M2YK2S0Z
kind: event
event_kind: finding
created: 2026-07-22T16:16:04Z
created_by: a-0htd7wjqyt
about: [[t-01KY59FNFAEE0KT7PWV8HAAY4A]]
origin: agent
applied: true
---
mdstore.WriteFile leaks its temp file when os.Rename fails

internal/mdstore/mdstore.go:471 — WriteFile's final line is 'return os.Rename(name, path)' with NO os.Remove(name) on the error path, unlike the WriteString path (:463-465) and Close path (:467-469) which both os.Remove the temp on failure. A rename failure (cross-device link, a dir replaced mid-write, EACCES, index lock) therefore orphans a '.dacli-tmp-*' file in the object directory. Every workspace write funnels through here (tasks, notes, events, agents), so a transient rename fault litters the tree with temp files that no later read/write cleans up. Confirms the last clause of sibling [[f-swallowed-i-o-errors-readdir-failures-reported-as-empty-run-record-writes-discarded-writefile-orphans-its-temp-file-on-rename-failure]] — STILL PRESENT. Fix: defer os.Remove(name) guarded by a success flag, or os.Remove on the rename error branch.
