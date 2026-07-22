---
id: d-ghmirror-recovery-uses-the-strongly-consistent-issue-list-endpoint-exact
kind: note
note_kind: decision
created: 2026-07-22T16:29:44Z
created_by: a-sfa41hsara
about: [[053]]
---
# ghmirror recovery uses the strongly-consistent issue-list endpoint + exact substring match, not the eventually-consistent search index
## Chose
ghmirror recovery uses the strongly-consistent issue-list endpoint + exact substring match, not the eventually-consistent search index
## Rejected
keep gh issue list --search and only soften the docstring's zero-duplicate claim
## Because
the finding f-ghmirror-marker-idempotency shows --search hits GitHub's eventually-consistent, tokenized search index, so a fast retry after a create-then-crash finds nothing and duplicates — exactly the failure the docstring promises to prevent; fetching bodies via the plain list endpoint and matching the marker by byte-substring makes the zero-duplicate guarantee actually hold, so the fixture is updated to the new contract
