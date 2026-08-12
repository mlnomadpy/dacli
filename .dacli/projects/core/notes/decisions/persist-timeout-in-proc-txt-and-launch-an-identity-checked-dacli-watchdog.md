---
id: d-persist-timeout-in-proc-txt-and-launch-an-identity-checked-dacli-watchdog
kind: note
note_kind: decision
created: 2026-08-12T14:01:38Z
created_by: a-codex-maintainer-nhx5wh
about: "[[372]]"
github:
  issue: 502
  repo: mlnomadpy/dacli
---
# Persist timeout in proc.txt and launch an identity-checked dacli watchdog
## Chose
Persist timeout in proc.txt and launch an identity-checked dacli watchdog
## Rejected
Keep timeout enforcement in the launching process or kill by numeric PGID alone
## Because
A launcher-local goroutine/context dies when detached spawn exits, while a bare PGID can be recycled. A separate no-pipe watchdog survives the launcher and revalidates the recorded PID start identity immediately before group signalling.
