---
id: f-ghmirror-marker-idempotency-can-still-duplicate-recovery-leans-on-eventually
kind: note
note_kind: finding
created: 2026-07-22T16:17:27Z
created_by: a-9y38s7w8e2
about: [[t-01KY59FNFK27A1084PQ8R2CJ5S]]
source_event: 01KY59QXYZRND966AN1CWK7NM5
---
# ghmirror marker-idempotency can still duplicate: recovery leans on eventually-consistent GitHub search
ghmirror.go:1-10 docstring asserts the load-bearing property: 'a retried sync after a timeout must converge with ZERO duplicate issues — the characteristic failure of naive syncers.' The mapping write-back (ghmirror.go:196) covers the normal path, but the crash-recovery path (create succeeded, local mapping write did not) relies on searchByMarker (ghmirror.go:174, 233-245) via 'gh issue list --search <marker>'. GitHub's issue search index is EVENTUALLY consistent (seconds-to-minutes lag) and tokenizes/strips punctuation, so a fast retry right after a create-then-crash can find zero hits and create a DUPLICATE — exactly the failure the docstring claims to prevent. Additionally the marker '<!-- dacli:ULID ws:WSID -->' (ghmirror.go:216) is angle-bracketed/colon-laden; --search matches word tokens, not substrings, so match reliability is unproven. Either soften the ZERO-duplicate claim to 'converges once indexing catches up', or gate creation behind a confirm-by-refetch, or verify --search actually matches the marker token end-to-end (unverified here — no network in sandbox).
