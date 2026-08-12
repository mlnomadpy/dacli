---
id: f-detached-spawn-discarded-its-recorded-timeout-after-launch
kind: note
note_kind: finding
created: 2026-08-12T14:01:38Z
created_by: a-codex-maintainer-nhx5wh
about: "[[372]]"
severity: major
---
# Detached spawn discarded its recorded timeout after launch
internal/features/execution/execution.go detached execRuntime branch used exec.Command with no deadline and returned immediately; its own comment said timeout enforcement required manual dacli kill/watchdog, but cmdSpawn started no watchdog. timeout_s in invocation.txt therefore could not terminate a silent detached process tree after the launcher exited.
