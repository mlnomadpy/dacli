---
id: 01KZVAJ6JX09FKGTXPCMV6RV7V
kind: event
event_kind: commit
created: 2026-08-12T15:48:25Z
created_by: a-codex-maintainer-3vy9w1
about: "[[t-01KZV1GAHS8M803TX5M22KRMFV]]"
origin: agent
applied: true
---
e836ae3 370: keep loop dry-run side-effect free

Regression red before fix:
TestLoopDryRunLeavesWorkspaceAndGovernorUntouched: dry-run 1 fed the production thrash guard: no net progress for 3 consecutive cycles — thrash guard tripped

Dry-run now skips persistent checkpoints, direct reconciliation/reaping/stage mutation, and governor charging while retaining the real phase planning path.
role: codex-maintainer
