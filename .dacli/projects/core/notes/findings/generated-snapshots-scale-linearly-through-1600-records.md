---
id: f-generated-snapshots-scale-linearly-through-1600-records
kind: note
note_kind: finding
created: 2026-08-20T08:13:17Z
created_by: a-maintainer-p5kmb7
about: "[[t-01M0AEG5K7JF96HV0RJ5K17NJN]]"
severity: moderate
---
# Generated snapshots scale linearly through 1600 records
Apple M5 Pro, benchtime=1x: tasks 100/400/1600 = 1.60/6.28/26.36ms and 0.46/1.81/7.28MB; events = 7.58/20.05/58.71ms and 0.96/3.81/15.28MB. Pure loaded brief render = 1.60ms, 0.50MB, 2,253 allocs.
