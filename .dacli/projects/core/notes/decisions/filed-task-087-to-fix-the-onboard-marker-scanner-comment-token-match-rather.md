---
id: d-filed-task-087-to-fix-the-onboard-marker-scanner-comment-token-match-rather
kind: note
note_kind: decision
created: 2026-07-23T00:00:53Z
created_by: a-48ab0df8g5
about: [[084]]
---
# Filed task 087 to fix the onboard marker scanner (comment-token match) rather than the DACLI_AGENT test-isolation gap or the blocked-051 merge conflict
## Chose
Filed task 087 to fix the onboard marker scanner (comment-token match) rather than the DACLI_AGENT test-isolation gap or the blocked-051 merge conflict
## Rejected
file the DACLI_AGENT test-env leak (f-021 already added a TestMain clearing it) or a merge-conflict-resolution task for blocked 051 (already tracked as blocked, owned by a-root)
## Because
the marker false-positive is directly observable in this very brief's Codebase map, degrades EVERY brief (not one test run), and actively corrupts adopt --todos seeding — the highest reader-facing leverage of the available leads; the DACLI_AGENT gap is already mitigated and 051 is a known tracked merge, not a new evidence-based improvement
