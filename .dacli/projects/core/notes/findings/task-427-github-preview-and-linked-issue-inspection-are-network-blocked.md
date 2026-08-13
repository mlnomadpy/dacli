---
id: f-task-427-github-preview-and-linked-issue-inspection-are-network-blocked
kind: note
note_kind: finding
created: 2026-08-13T19:10:33Z
created_by: a-codex-maintainer-mjejj8
about: "[[427]]"
severity: moderate
---
# task 427 GitHub preview and linked issue inspection are network-blocked
Required /private/tmp/dacli-loop-current github push core 427 --dry-run failed before producing a preview because gh repo view could not connect to api.github.com. The linked GitHub issue therefore could not be read from this environment; local task brief and recorded 422/423 commit evidence were used.
