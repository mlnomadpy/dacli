---
id: 01KY85KC0R0KMJN9GFXR7J20YT
kind: event
event_kind: finding
created: 2026-07-23T19:00:24Z
created_by: a-nfazzjdrh2
about: [[t-01KY60QM1Y7DK05WXB954YNDHJ]]
origin: agent
applied: false
---
mdstore SetList round-trip is lossy for elements holding both quote chars plus a comma

quoteListElem (internal/mdstore/mdstore.go:153-161) wraps an element in double quotes whenever the single-quote branch is skipped, but never ESCAPES an embedded double quote. An element containing a ' (forces the double-quote branch), a ", and a , -- e.g. it's "a,b" -- is emitted by SetList (mdstore.go:140-146) as ["it's "a,b""]. On read-back GetList->splitTop (mdstore.go:203) sees the embedded " close the wrapping quote, then the following , splits at depth 0, so GetList returns 2 corrupted elements instead of 1. This violates mdstore invariant #1 (lossless round-trip, mdstore.go:4-9) and the SetList doc comment's 'round-trips losslessly' claim (mdstore.go:136-139). TestSetListRoundTrip (mdstore_test.go:233-258) covers each quote type in isolation and 'with,comma' but never an element holding both ' and " together with a comma, so CI is green while the invariant is broken. Consumers: runtimefiles SandboxRO/Args/Env and gates route through SetList (task 119), so a runtime arg/env value like a message template carrying an apostrophe + a quoted phrase + a comma silently loses fidelity.
