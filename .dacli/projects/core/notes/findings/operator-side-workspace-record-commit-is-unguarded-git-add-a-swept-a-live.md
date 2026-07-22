---
id: f-operator-side-workspace-record-commit-is-unguarded-git-add-a-swept-a-live
kind: note
note_kind: finding
created: 2026-07-22T15:04:04Z
created_by: a-root
severity: major
---
# Operator-side workspace-record commit is unguarded: git add -A swept a live worktree as an embedded gitlink (this workspace's .gitignore also lacked worktrees/). E2 guards AGENT commits by claim, but the operator's record commit has no dacli-native path — it should stage only agents/roles/projects/events/notes and never worktrees/runs/build
