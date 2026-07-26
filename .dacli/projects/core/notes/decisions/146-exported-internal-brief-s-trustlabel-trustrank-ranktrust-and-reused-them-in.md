---
id: d-146-exported-internal-brief-s-trustlabel-trustrank-ranktrust-and-reused-them-in
kind: note
note_kind: decision
created: 2026-07-26T15:21:48Z
created_by: a-94k1g003b6
---
# 146: exported internal/brief's TrustLabel/TrustRank/RankTrust and reused them in vcs, instead of duplicating the refuted<unverified<confirmed ordering
## Chose
146: exported internal/brief's TrustLabel/TrustRank/RankTrust and reused them in vcs, instead of duplicating the refuted<unverified<confirmed ordering
## Rejected
duplicating the 3 small rank/label switch statements locally in vcs.go the way verdictMarker duplicates execution.VerdictMarker
## Because
internal/brief is not a feature slice (TestFeatureSlicesAreIsolated only forbids features/* importing features/*), so importing it from vcs is architecturally legal and adds no import cycle; a second copy of the trust-ordering rule would silently drift from brief's D3 trust-floor the next time someone tweaks grading semantics, whereas one exported source of truth keeps the PR's trust grade and the brief's trust-floor permanently in lockstep
