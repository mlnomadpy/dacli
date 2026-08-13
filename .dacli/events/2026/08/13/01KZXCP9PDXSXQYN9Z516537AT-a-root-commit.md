---
id: 01KZXCP9PDXSXQYN9Z516537AT
kind: event
event_kind: commit
created: 2026-08-13T11:04:05Z
created_by: a-root
about: "[[t-01KZX7PXQBEVM1M0N2BKWYD4RK]]"
origin: agent
applied: true
---
b6d2b1f 406: classify detached provider failures

Persist the guardian's runtime exit code so wait can classify detached CLI
failures, open the same durable cooldown, and print/record the transition.
Also satisfy the pinned lint contract for the policy core.

Red: TestRunGuardianPersistsRuntimeExitCode failed with runtime-exit.txt: no
such file or directory when the guardian exit-file write was disabled.
role: root
