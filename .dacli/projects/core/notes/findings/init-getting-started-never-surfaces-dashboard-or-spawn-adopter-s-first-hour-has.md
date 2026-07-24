---
id: f-init-getting-started-never-surfaces-dashboard-or-spawn-adopter-s-first-hour-has
kind: note
note_kind: finding
created: 2026-07-24T09:31:57Z
created_by: a-3k7h1pghgd
about: [[139]]
severity: moderate
---
# init 'Getting started' never surfaces dashboard or spawn — adopter's first hour has no pointer to the fleet UI
printGettingStarted (internal/features/wscore/wscore.go:87-106) lists exactly 5 steps: whoami, project add, task add, next, overview. Neither 'dacli dashboard' (internal/features/dashboard/dashboard.go:31) nor 'spawn' appears. A new adopter running init is never told the fleet UI exists, so first-run mental model is 'a task tracker', not 'an agent swarm' — the exact expectation gap probed by INTERVIEW_GUIDE.md §4 Q2/Q3. Grounds the adopter interview's first-run confusion answers.
