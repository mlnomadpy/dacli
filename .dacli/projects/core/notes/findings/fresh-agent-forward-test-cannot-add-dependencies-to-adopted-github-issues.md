---
id: f-fresh-agent-forward-test-cannot-add-dependencies-to-adopted-github-issues
kind: note
note_kind: finding
created: 2026-08-19T12:14:46Z
created_by: a-maintainer-ebqr9f
about: "[[t-01M0CX031NDQ5PQ8VRX1PQNWXE]]"
severity: major
---
# Fresh-agent forward test cannot add dependencies to adopted GitHub issues
GitHub pull creates task records without dependency edges in internal/features/ghmirror/ghmirror.go around line 1044, while task add is the only command exposing depends-on in internal/features/planning/planning.go. A fresh agent therefore cannot execute the documented pull then dependency-design flow through the CLI. The representative forward test stopped at this boundary.
