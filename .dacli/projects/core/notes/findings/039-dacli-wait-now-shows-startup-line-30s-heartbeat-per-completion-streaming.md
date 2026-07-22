---
id: f-039-dacli-wait-now-shows-startup-line-30s-heartbeat-per-completion-streaming
kind: note
note_kind: finding
created: 2026-07-22T15:31:38Z
created_by: a-87qa0eamp1
about: [[039]]
severity: minor
---
# 039: dacli wait now shows startup line + 30s heartbeat, per-completion streaming unchanged
cmdWait (internal/features/execution/execution.go ~:1347) now prints 'waiting on N run(s): <short ids>' at startup and, in the poll loop, a light heartbeat 'still waiting on K run(s) (up <elapsed>)' every ~30s (nextBeat timer, not every poll). Per-completion line gained '(N of M)' progress and still prints the moment each child is detected gone via finalizeRun. go build ./... clean; go test ./internal/... all green.
