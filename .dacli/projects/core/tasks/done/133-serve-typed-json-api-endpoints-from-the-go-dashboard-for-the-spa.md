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
- [x] dacli dashboard serves /api/overview, /api/projects, /api/tasks, /api/agents as JSON (the data the current page renders), documented shape, one handler test per endpoint
- [x] endpoints reflect the live event log + store so agents update in near real time; existing dashboard behavior preserved until the SPA replaces it
## Log
- 2026-07-24T00:49:44Z claimed by a-wey686j4cx
- 2026-07-24T00:55:37Z accepted by a-root
- 2026-07-24T00:55:37Z completed by a-root
