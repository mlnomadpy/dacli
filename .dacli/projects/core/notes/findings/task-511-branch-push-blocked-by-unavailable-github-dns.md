---
id: f-task-511-branch-push-blocked-by-unavailable-github-dns
kind: note
note_kind: finding
created: 2026-08-28T00:44:24Z
created_by: a-fixer-0mrzjc
about: "[[t-01M1068MTFPQ6YFVQG204M2EX4]]"
severity: major
---
# Task 511 branch push blocked by unavailable GitHub DNS
After commit 16dda175, /tmp/dacli-main push --task t-01M1068MTFPQ6YFVQG204M2EX4 returned fatal: unable to access https://github.com/mlnomadpy/dacli.git/: Could not resolve host: github.com. The headless sandbox cannot open the required PR.
