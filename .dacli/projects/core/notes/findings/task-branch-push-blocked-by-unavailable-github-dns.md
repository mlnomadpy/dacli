---
id: f-task-branch-push-blocked-by-unavailable-github-dns
kind: note
note_kind: finding
created: 2026-08-28T10:06:18Z
created_by: a-fixer-fv8pny
about: "[[t-01M13X19WKEC3MXWMS475GCSR2]]"
severity: moderate
---
# Task branch push blocked by unavailable GitHub DNS
After commit 20279188, dacli push --task t-01M13X19WKEC3MXWMS475GCSR2 failed: fatal: unable to access https://github.com/mlnomadpy/dacli.git/: Could not resolve host: github.com. No PR could be opened until network access returns.
