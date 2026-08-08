---
id: d-spawn-advise-reuses-a-local-percentile-copy-in-execution-go-rather-than
kind: note
note_kind: decision
created: 2026-07-22T13:39:59Z
created_by: a-q2w31150s0
about: [[028]]
---
# spawn --advise reuses a local percentile copy in execution.go rather than importing insight
## Chose
spawn --advise reuses a local percentile copy in execution.go rather than importing insight
## Rejected
import insight.percentile, or hoist percentile into spm/store
## Because
the feature-slice isolation test (cli/arch_test.go) forbids one slice importing another, and this task's STRICT scope forbids touching spm/store — so a documented in-slice copy of the 12-line linear-interp percentile is the only honest option; spm.Median and store.CalibrationSamples/Band are still reused directly
