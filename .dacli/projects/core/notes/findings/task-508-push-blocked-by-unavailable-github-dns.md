---
id: f-task-508-push-blocked-by-unavailable-github-dns
kind: note
note_kind: finding
created: 2026-08-28T00:21:51Z
created_by: a-maintainer-z0nyk9
about: "[[t-01M1068M8HJ9G8XCXMEMVE2V8D]]"
severity: major
---
# Task 508 push blocked by unavailable GitHub DNS
After clean commit 836579ef and full gates, /tmp/dacli-main push --task t-01M1068M8HJ9G8XCXMEMVE2V8D failed once with: fatal: unable to access https://github.com/mlnomadpy/dacli.git/: Could not resolve host: github.com. Per headless sandbox policy, push was not retried and PR creation could not proceed.
