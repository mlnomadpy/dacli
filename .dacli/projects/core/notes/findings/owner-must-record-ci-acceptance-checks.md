---
id: f-owner-must-record-ci-acceptance-checks
kind: note
note_kind: finding
created: 2026-08-28T10:06:00Z
created_by: a-fixer-fv8pny
about: "[[t-01M13X19WKEC3MXWMS475GCSR2]]"
severity: minor
---
# Owner must record CI acceptance checks
After full go test ./... passed, dacli task check t-01M13X19WKEC3MXWMS475GCSR2 --n 1 returned exit-3 refusal: only owner a-root checks acceptance boxes. Criteria 1–4 have local evidence; criterion 5 remains blocked by GitHub API DNS.
