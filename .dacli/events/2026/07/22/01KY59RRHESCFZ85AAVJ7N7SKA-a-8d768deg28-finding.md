---
id: 01KY59RRHESCFZ85AAVJ7N7SKA
kind: event
event_kind: finding
created: 2026-07-22T16:15:32Z
created_by: a-8d768deg28
about: [[t-01KY59FNF0CRTHD6SECSM2ZC6H]]
origin: agent
applied: true
---
Network.Parallelizable claims dependency-satisfied filtering it cannot perform

criticalpath.go:259-291 Parallelizable's doc claims it returns 'tasks whose dependencies are already satisfied, ordered critical-path first'. But Network carries no edge/adjacency data (only Duration, Schedules, CriticalPath), so the body only excludes done[] tasks and sorts by slack (criticalpath.go:271-282) — it never consults dependencies. A zero-slack task whose predecessor is not yet done is still returned as 'worth spawning subagents on right now'. It has no production caller (only spm_test.go), so impact is low, but the contract is false. Either drop the dependency claim from the doc or pass edges/a satisfied-set so the filter is real. cmdNext (insight.go:148) reimplements readiness correctly and does not use this helper.
