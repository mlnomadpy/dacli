---
id: d-area-label-last-dir-segment-of-first-internal-path-strip-stale-severity-labels
kind: note
note_kind: decision
created: 2026-07-22T19:50:21Z
created_by: a-jfsgjqqya0
about: [[070]]
---
# area label = last dir segment of first internal path; strip stale severity labels on re-push
## Chose
area label = last dir segment of first internal path; strip stale severity labels on re-push
## Rejected
map only the correct severity label without removing stale ones; segment right after internal/ as area
## Because
public-repo severity:unspecified persists on issues filed before the note carried severity, so a fix must STRIP the stale label (mirroring otherStatusLabels), not merely add the correct one; leaf package (ghmirror in internal/features/ghmirror, store in internal/store) is the meaningful slice so LAST dir segment names the area, while the segment after internal/ would wrongly yield features
