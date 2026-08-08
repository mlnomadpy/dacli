---
id: f-124-docs-index-hero-uses-the-real-dashboard-png-not-a-placeholder
kind: note
note_kind: finding
created: 2026-07-23T21:49:52Z
created_by: a-ksvdbbt934
about: [[124]]
severity: minor
---
# 124: docs index hero uses the real dashboard.png, not a placeholder
docs/assets/dashboard.png already existed (real 1600x900 screenshot, committed d57ff48 by the task-122 agent) at the time this task ran, so docs/index.md:17-21 wires an img tag to assets/dashboard.png directly with the caption mission control -- the live agent swarm -- no fabricated image, no placeholder comment needed.
