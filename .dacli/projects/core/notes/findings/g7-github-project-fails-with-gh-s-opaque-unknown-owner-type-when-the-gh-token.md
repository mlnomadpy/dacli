---
id: f-g7-github-project-fails-with-gh-s-opaque-unknown-owner-type-when-the-gh-token
kind: note
note_kind: finding
created: 2026-07-22T21:31:03Z
created_by: a-root
severity: minor
---
# G7 github project fails with gh's opaque 'unknown owner type' when the gh token lacks the 'project' scope (Projects v2 needs it separately from repo). Detect the missing scope and tell the operator to run 'gh auth refresh -s project' instead of surfacing gh's cryptic error
