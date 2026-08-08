---
id: d-pull-is-inbound-so-it-is-not-gated-on-public-visibility-only-outbound-push
kind: note
note_kind: decision
created: 2026-07-22T17:35:03Z
created_by: a-sdpxn53045
about: [[057]]
---
# Pull is inbound so it is NOT gated on public visibility; only outbound push + finding comments trip the disclosure gate
## Chose
Pull is inbound so it is NOT gated on public visibility; only outbound push + finding comments trip the disclosure gate
## Rejected
Apply the disclosure gate to pull too, for literal 'same gate as push' compliance
## Because
Pulling imports an issue INTO the workspace — it discloses nothing, and a public-refusal on pull would block the common case (importing from a public repo), directly contradicting the acceptance criterion 'dacli can import a GitHub issue as a task'. The factored disclosureGate() helper is shared by push and its finding-comment path (the risk-rank-2 leak surface); pull only requires the project be linked.
