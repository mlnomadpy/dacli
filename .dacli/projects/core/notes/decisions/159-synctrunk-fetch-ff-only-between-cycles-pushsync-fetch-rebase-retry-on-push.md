---
id: d-159-synctrunk-fetch-ff-only-between-cycles-pushsync-fetch-rebase-retry-on-push
kind: note
note_kind: decision
created: 2026-07-26T22:05:27Z
created_by: a-kbvnat1ma2
about: [[159]]
---
# 159: syncTrunk (fetch+ff-only) between cycles + PushSync (fetch+rebase retry) on push rejection, rather than a merge-based reconciliation
## Chose
159: syncTrunk (fetch+ff-only) between cycles + PushSync (fetch+rebase retry) on push rejection, rather than a merge-based reconciliation
## Rejected
reconciling via git merge (not ff-only/rebase) of origin into local main between cycles
## Because
a merge would create a spurious merge commit on every cycle boundary even when nothing diverged, and would need the same dirty-tree tolerance gitx.Merge already special-cases for .dacli churn; ff-only is a no-op when already caught up and only ever fast-forwards (never risks a merge commit or conflict resolution on trunk itself), while the one real divergence case -- local has an unpushed record commit while origin gained an async auto-merge -- is exactly what the push-time rebase (gitx.PushSync) exists to reconcile at the moment it actually matters (the push), not speculatively every cycle
