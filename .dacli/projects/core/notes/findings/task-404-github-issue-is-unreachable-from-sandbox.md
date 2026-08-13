---
id: f-task-404-github-issue-is-unreachable-from-sandbox
kind: note
note_kind: finding
created: 2026-08-13T09:46:51Z
created_by: a-codex-maintainer-2hqkmd
about: "[[404]]"
severity: moderate
---
# Task 404 GitHub issue is unreachable from sandbox
Local task links mlnomadpy/dacli issue #548. Running 'gh issue view 548 --json number,title,body,url' failed with 'error connecting to api.github.com'; implementation therefore uses the local task and repository specifications, with no remote mutation.
