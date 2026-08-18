---
id: f-role-removal-swallowed-unreadable-roster-and-run-state
kind: note
note_kind: finding
created: 2026-08-18T12:56:20Z
created_by: a-maintainer-ytrsg6
about: "[[t-01M0AETPE835JWHHS5GA5RE4AW]]"
severity: major
---
# Role removal swallowed unreadable roster and run state
internal/store/remove.go previously ignored ListAgents errors, while internal/store/roles.go ListAgents and run scans skipped individual read failures; role rm could therefore delete a capability after certifying an unreadable holder/run as absent. Focused regression TestRemoveRoleFailsClosedOnUnreadableState reproduces both shapes.
