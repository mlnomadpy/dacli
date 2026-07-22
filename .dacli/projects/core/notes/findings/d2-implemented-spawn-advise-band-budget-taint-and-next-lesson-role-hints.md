---
id: f-d2-implemented-spawn-advise-band-budget-taint-and-next-lesson-role-hints
kind: note
note_kind: finding
created: 2026-07-22T13:40:11Z
created_by: a-q2w31150s0
about: [[028]]
severity: moderate
---
# D2 implemented: spawn --advise (band budget + taint) and next lesson→role hints
execution.go cmdSpawn: added --advise flag; after role/model/runtime/task resolve but BEFORE agentid.Spawn, printAdvisory() prints (1) a suggested budget/sizing from store.CalibrationSamples matching store.Band{OrDash(role),OrDash(model),rt.Name} when the band has n>=10 (D1 threshold), else 'no band history yet' — median x-ratio projected to hours via Te, labelled wall-clock proxy; (2) taint status via store.Taint('external:') + TaintResult.ExposedBriefs — 'task NNN is in the blast radius of <origins>' if the task slug is exposed, else 'taint: clean'. Advice is additive; spawn proceeds unchanged (axiom 3). insight.go cmdNext: after the --parallel candidate list, scope-matched store.WorkspaceLessons annotate shown tasks with 'lesson L applies — consider role R', where R is the role whose scope glob covers a path the lesson cites (else a role named in the lesson). Band built in recorded OrDash form so it matches invocation.txt bands. percentile is a documented local copy (arch_test forbids execution->insight import; strict scope forbids a shared helper). go build + go test ./internal/... green incl. TestFeatureSlicesAreIsolated.
