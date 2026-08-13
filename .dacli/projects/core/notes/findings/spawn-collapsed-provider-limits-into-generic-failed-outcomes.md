---
id: f-spawn-collapsed-provider-limits-into-generic-failed-outcomes
kind: note
note_kind: finding
created: 2026-08-13T10:20:24Z
created_by: a-codex-maintainer-76ksyq
about: "[[406]]"
severity: major
---
# Spawn collapsed provider limits into generic failed outcomes
internal/features/execution/execution.go previously returned every nonzero runtime exit as child failed without typed classification or a durable cooldown; the new runtimepolicy boundary records cooldown state under runs/runtime-cooldowns before subsequent resolveLaunch calls.
