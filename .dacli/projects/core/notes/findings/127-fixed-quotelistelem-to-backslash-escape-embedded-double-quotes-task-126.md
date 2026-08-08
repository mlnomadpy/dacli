---
id: f-127-fixed-quotelistelem-to-backslash-escape-embedded-double-quotes-task-126
kind: note
note_kind: finding
created: 2026-07-26T21:07:55Z
created_by: a-mh7sb8yq7e
about: [[127]]
severity: moderate
---
# 127: fixed quoteListElem to backslash-escape embedded double quotes; task 126 (same title) had been closed done with no actual code change to mdstore.go
internal/mdstore/mdstore.go:153-186: quoteListElem now always double-quote-wraps when quoting is needed and escapes embedded double-quote and backslash characters with a leading backslash; clean() and splitTop() skip escaped characters while scanning for the closing quote, and a new unescapeDouble() reverses the escaping after stripping the outer quotes. Added TestSetListRoundTrip cases for an element containing both quote chars plus a comma (the literal its "a,b"), a mixed multi-element case, and elements with backslashes. go build ./... and go test ./... both green. Verified via git log that task 126 (closed done, empty acceptance) never touched mdstore.go -- the bug it claimed to fix was still present before this change.
