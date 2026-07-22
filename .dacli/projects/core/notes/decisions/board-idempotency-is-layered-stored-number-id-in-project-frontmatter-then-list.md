---
id: d-board-idempotency-is-layered-stored-number-id-in-project-frontmatter-then-list
kind: note
note_kind: decision
created: 2026-07-22T20:15:15Z
created_by: a-rncv68m0fd
about: [[071]]
---
# board idempotency is layered: stored number+id in project frontmatter, then list-by-title adoption, then create; items keyed by content issue-number snapshot
## Chose
board idempotency is layered: stored number+id in project frontmatter, then list-by-title adoption, then create; items keyed by content issue-number snapshot
## Rejected
add every issue every run and rely on gh project item-add to dedupe
## Because
gh project item-add does NOT dedupe by content, so a naive re-run would stack duplicate board items — the acceptance's 're-run does not duplicate items'; instead ensureItem consults a one-shot item-list snapshot (itemIndexByNumber: content issue-number → item id) and only adds an issue absent from it, and ensureProject resolves the board by stored github_project block (number+id, both required) before a list-by-title adoption before create, so a lost write-back is recovered by title rather than duplicated — mirrors the existing task/decision/finding marker idempotency in ghmirror.go
