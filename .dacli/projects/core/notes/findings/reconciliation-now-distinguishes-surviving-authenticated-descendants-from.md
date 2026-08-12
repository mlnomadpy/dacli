---
id: f-reconciliation-now-distinguishes-surviving-authenticated-descendants-from
kind: note
note_kind: finding
created: 2026-08-12T14:24:32Z
created_by: a-codex-maintainer-r0c643
about: "[[375]]"
severity: major
---
# Reconciliation now distinguishes surviving authenticated descendants from recycled groups
internal/procmon/procmon.go:189 requires PID=PGID plus PIDStart for dead-leader fallback and rechecks the leader around GroupAlive; focused TestRunStillLivePreservesTask177AfterLeaderExit and task-369 liveness/kill controls pass.
