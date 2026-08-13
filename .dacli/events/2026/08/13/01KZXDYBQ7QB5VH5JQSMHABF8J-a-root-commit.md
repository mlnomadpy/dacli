---
id: 01KZXDYBQ7QB5VH5JQSMHABF8J
kind: event
event_kind: commit
created: 2026-08-13T11:25:58Z
created_by: a-root
about: "[[t-01KZX7PXQBEVM1M0N2BKWYD4RK]]"
origin: agent
applied: true
---
78054b3 406: isolate guardian exit test process group

Run the guardian regression through the package test binary in its own process
group, matching production. Direct invocation shared the race suite's group and
correctly waited for unrelated processes until CI's ten-minute timeout.
role: root
