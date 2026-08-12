---
id: d-authenticate-dead-leader-descendants-with-the-recorded-leader-pid-start-pair
kind: note
note_kind: decision
created: 2026-08-12T14:14:35Z
created_by: a-codex-maintainer-1ecns6
about: "[[375]]"
github:
  issue: 494
  repo: mlnomadpy/dacli
---
# Authenticate dead-leader descendants with the recorded leader PID/start pair
## Chose
Authenticate dead-leader descendants with the recorded leader PID/start pair
## Rejected
Trust any live numeric PGID, or add a permanent sentinel process
## Because
Every spawned group records PID=PGID and PIDStart. A genuine surviving group has the original PID absent; a recycled group must have a new live group leader at that same PID whose start identity mismatches. Requiring the durable identity and rechecking the PID around GroupAlive preserves task 177 without signalling task 369's stranger.
