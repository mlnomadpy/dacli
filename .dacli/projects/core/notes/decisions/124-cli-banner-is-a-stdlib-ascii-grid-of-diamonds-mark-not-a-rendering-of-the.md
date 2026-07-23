---
id: d-124-cli-banner-is-a-stdlib-ascii-grid-of-diamonds-mark-not-a-rendering-of-the
kind: note
note_kind: decision
created: 2026-07-23T21:49:46Z
created_by: a-ksvdbbt934
about: [[124]]
---
# 124: CLI banner is a stdlib ASCII grid-of-diamonds mark, not a rendering of the SVG logo
## Chose
124: CLI banner is a stdlib ASCII grid-of-diamonds mark, not a rendering of the SVG logo
## Rejected
shelling out to an SVG-to-ASCII renderer, or a figlet-style wordmark
## Because
the brief requires stdlib string art only, no new deps; a hand-authored diamond grid (docs/assets/logo.svg's hexagon-cluster mark, reinterpreted for a terminal) carries the same 'coordinated units, not chaos' motif without any rendering dependency
