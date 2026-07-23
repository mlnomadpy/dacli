---
id: t-01KY8KW3X543JFEMD2DHB6SFEC
kind: task
created: 2026-07-23T23:09:51Z
created_by: a-root
owner: a-root
priority: must
depends_on: [132]
---
# Serve typed JSON API endpoints from the Go dashboard for the SPA
## So that
the Vue app reads real data over a stable, tested contract
## Acceptance
- [ ] dacli dashboard serves /api/overview, /api/projects, /api/tasks, /api/agents as JSON (the data the current page renders), documented shape, one handler test per endpoint
- [ ] endpoints reflect the live event log + store so agents update in near real time; existing dashboard behavior preserved until the SPA replaces it
## Log
