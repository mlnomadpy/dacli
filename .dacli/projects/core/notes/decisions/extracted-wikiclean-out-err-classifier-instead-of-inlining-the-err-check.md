---
id: d-extracted-wikiclean-out-err-classifier-instead-of-inlining-the-err-check
kind: note
note_kind: decision
created: 2026-08-04T11:40:04Z
created_by: a-maintainer-f89wdf
about: "[[219]]"
---
# extracted wikiClean(out,err) classifier instead of inlining the err check
## Chose
extracted wikiClean(out,err) classifier instead of inlining the err check
## Rejected
inline 'out, err := git(...); if err != nil { return }' at the call site like siblings add/commit/push
## Because
the failed-status-vs-clean-tree decision is the whole bug and it is untestable inside publishWiki (which network-clones a real wiki); a tiny pure classifier lets a unit test pin the contract (err -> error, empty+no-err -> clean, non-empty -> dirty) deterministically, matching this package's existing pure-helper tests, while the call site reads identically to the other checked git calls
