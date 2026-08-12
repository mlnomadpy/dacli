---
id: 01KZV9S09DDYZGM5D2HJXCNWKK
kind: event
event_kind: commit
created: 2026-08-12T15:34:39Z
created_by: a-root
about: "[[t-01KZV5RSPJ5PSG74VHH2Q4A7JQ]]"
origin: agent
applied: true
---
6ee09c4 379: migrate legacy halted loop marker

Use the companion loop-status marker when a pre-379 governor snapshot has no marker fields, preserving cycle and token accounting while allowing a real intervening trunk advance to clear the thrash streak.

Regression: TestLoopRestartMigratesLegacyHaltMarkerFromLoopStatus.
role: root
