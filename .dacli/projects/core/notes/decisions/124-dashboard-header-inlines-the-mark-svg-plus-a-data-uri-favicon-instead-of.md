---
id: d-124-dashboard-header-inlines-the-mark-svg-plus-a-data-uri-favicon-instead-of
kind: note
note_kind: decision
created: 2026-07-23T21:49:48Z
created_by: a-ksvdbbt934
about: [[124]]
---
# 124: dashboard header inlines the mark SVG plus a data-URI favicon instead of adding a static-asset route
## Chose
124: dashboard header inlines the mark SVG plus a data-URI favicon instead of adding a static-asset route
## Rejected
wiring docs/assets/favicon.svg through a new /assets/ HTTP route on the dashboard server
## Because
internal/features/dashboard only go:embeds static/index.html today (no static-file route exists); inlining the same polygon coordinates gets the identical mark on the dashboard header with zero new server surface, matching the brief's 'where trivial' scope for that surface
