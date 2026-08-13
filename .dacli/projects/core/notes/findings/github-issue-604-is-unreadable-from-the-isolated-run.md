---
id: f-github-issue-604-is-unreadable-from-the-isolated-run
kind: note
note_kind: finding
created: 2026-08-13T19:57:42Z
created_by: a-codex-maintainer-zkfgn1
about: "[[430]]"
severity: moderate
---
# GitHub issue 604 is unreadable from the isolated run
gh issue view 604 --repo mlnomadpy/dacli failed with error connecting to api.github.com; the local task and parent task log preserve the version/checksum/checkpoint requirement, but the public mirror could not be independently read.
