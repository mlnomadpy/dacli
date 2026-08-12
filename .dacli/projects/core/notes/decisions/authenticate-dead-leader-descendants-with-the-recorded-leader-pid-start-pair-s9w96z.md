---
id: d-authenticate-dead-leader-descendants-with-the-recorded-leader-pid-start-pair-s9w96z
kind: note
note_kind: decision
created: 2026-08-12T14:24:32Z
created_by: a-codex-maintainer-r0c643
about: "[[375]]"
github:
  issue: 493
  repo: mlnomadpy/dacli
---
# Authenticate dead-leader descendants with the recorded leader PID/start pair
## Chose
Authenticate dead-leader descendants with the recorded leader PID/start pair
## Rejected
Trust any live numeric PGID or add a permanent sentinel process
## Because
Every spawned group records PID=PGID and PIDStart. A genuine surviving group has the original PID absent; a recycled group must create a live leader at that PID. Requiring the durable identity and rechecking PID around GroupAlive preserves task 177 without signalling task 369's stranger.
