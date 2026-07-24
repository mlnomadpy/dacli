---
id: t-01KY8KW3Y5P8JSDSCYZQ89W1XR
kind: task
created: 2026-07-23T23:09:51Z
created_by: a-root
owner: a-root
priority: must
depends_on: [134]
---
# Embed the built SPA into the dacli binary and wire the frontend build into CI
## So that
dacli dashboard serves the SPA from one binary and the embedded assets stay fresh
## Acceptance
- [x] go:embed serves ui/dist from 'dacli dashboard' (with a documented dev-mode fallback); go build succeeds with embedded assets present
- [x] CI builds the frontend (npm ci && npm run build) before go build/release so embedded assets are current; ci.yml and the release workflow stay green
## Log
- 2026-07-24T01:24:41Z claimed by a-qn95ewfptr
- 2026-07-24T01:36:13Z accepted by a-root
- 2026-07-24T01:36:13Z completed by a-root
