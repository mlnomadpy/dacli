---
id: d-filed-the-mdstore-setlist-mixed-quote-comma-round-trip-bug-as-the-single
kind: note
note_kind: decision
created: 2026-07-23T19:02:14Z
created_by: a-nfazzjdrh2
about: [[084]]
---
# Filed the mdstore SetList mixed-quote+comma round-trip bug as the single highest-value evidence-based change
## Chose
Filed the mdstore SetList mixed-quote+comma round-trip bug as the single highest-value evidence-based change
## Rejected
Filing broader/speculative work (e.g. more of task 125's inline-list-writer routing, or an unverified sibling-finding follow-up), or a build/test-gated defect I could not confirm headlessly
## Because
This is a self-contained correctness bug I could verify by reading alone (concrete file:line at mdstore.go:153-161 and a concrete failing input, it's "a,b") that violates a DOCUMENTED core invariant (lossless round-trip, mdstore.go:4-9) and the SetList doc comment itself, yet passes CI because TestSetListRoundTrip only covers each quote type in isolation. It is not already covered by any open task (125 is about routing writers THROUGH the encoder, not fixing the encoder), the fix is small and testable, and it sits on a live consumer path (runtimefiles/gates via task 119). Higher evidentiary confidence + real blast radius + small scope beat broader speculative filings.
