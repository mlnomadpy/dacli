---
id: f-critical-path-reference-over-prescribes-auto-merge
kind: note
note_kind: finding
created: 2026-08-19T12:14:46Z
created_by: a-maintainer-ebqr9f
about: "[[t-01M0CX031NDQ5PQ8VRX1PQNWXE]]"
severity: moderate
---
# Critical-path reference over-prescribes auto-merge
skills/dacli/references/critical-path-github.md presents pr with auto as the minimum handoff, while github-landing.md conditions auto-merge on trustworthy required checks and reviews. A fresh agent should default to opening and observing a PR unless branch policy proves auto-merge safe.
