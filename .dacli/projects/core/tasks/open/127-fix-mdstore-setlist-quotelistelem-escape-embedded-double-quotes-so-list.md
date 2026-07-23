---
id: t-01KY85NK5SEZRCM3MDSSXT3KZJ
kind: task
created: 2026-07-23T19:01:37Z
created_by: a-nfazzjdrh2
owner: a-nfazzjdrh2
priority: should
---
# Fix mdstore SetList/quoteListElem: escape embedded double quotes so list elements holding both quote chars and a comma round-trip losslessly
## So that
runtimefiles SandboxRO/Args/Env and gates (routed through SetList by task 119) stop silently corrupting values that carry an apostrophe, a double quote, and a comma together — a documented lossless-round-trip invariant (mdstore.go:4-9) is currently violated; see the finding under task 084
## Acceptance
- [ ] quoteListElem (internal/mdstore/mdstore.go:153-161) escapes embedded double quotes (or otherwise guarantees the emitted element re-parses as exactly one element); clean/splitTop decode the escape so GetList is the exact inverse of SetList
- [ ] TestSetListRoundTrip (internal/mdstore/mdstore_test.go:233) gains cases for an element containing BOTH ' and " together with a comma (e.g. the literal it's "a,b") asserting len==1 and byte-equality; all existing cases stay green
- [ ] go build ./... clean and go test ./internal/mdstore/... green; no regression to any currently-passing round-trip case
## Log
