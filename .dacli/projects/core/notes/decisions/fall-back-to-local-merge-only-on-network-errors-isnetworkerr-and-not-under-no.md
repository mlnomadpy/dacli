---
id: d-fall-back-to-local-merge-only-on-network-errors-isnetworkerr-and-not-under-no
kind: note
note_kind: decision
created: 2026-07-22T21:42:10Z
created_by: a-c79p0msrw8
about: [[078]]
---
# Fall back to local merge ONLY on network errors (isNetworkErr), and NOT under --no-merge
## Chose
Fall back to local merge ONLY on network errors (isNetworkErr), and NOT under --no-merge
## Rejected
Fall back to local merge on any push/gh failure; or never fall back
## Because
The spec calls the offline fallback 'documented, not silent': a network-unreachable failure warns + local-merges so a wave lands offline, but a non-network failure (protected branch, bad auth, dirty tree) must surface — mislabeling it as offline would silently local-merge work that GitHub rejected. Under --no-merge the operator explicitly wanted PRs for human review, so an offline failure is surfaced as an error rather than merged behind their back.
