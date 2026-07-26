---
id: t-01KY85NK5SEZRCM3MDSSXT3KZJ
kind: task
created: 2026-07-23T19:01:37Z
created_by: a-nfazzjdrh2
owner: a-root
priority: should
github:
  issue: 223
  repo: mlnomadpy/dacli
---
# Fix mdstore SetList/quoteListElem: escape embedded double quotes so list elements holding both quote chars and a comma round-trip losslessly
## So that
runtimefiles SandboxRO/Args/Env and gates (routed through SetList by task 119) stop silently corrupting values that carry an apostrophe, a double quote, and a comma together — a documented lossless-round-trip invariant (mdstore.go:4-9) is currently violated; see the finding under task 084
## Acceptance
- [x] quoteListElem (internal/mdstore/mdstore.go:153-161) escapes embedded double quotes (or otherwise guarantees the emitted element re-parses as exactly one element); clean/splitTop decode the escape so GetList is the exact inverse of SetList
- [x] TestSetListRoundTrip (internal/mdstore/mdstore_test.go:233) gains cases for an element containing BOTH ' and " together with a comma (e.g. the literal it's "a,b") asserting len==1 and byte-equality; all existing cases stay green
- [x] go build ./... clean and go test ./internal/mdstore/... green; no regression to any currently-passing round-trip case
## Log
- 2026-07-26T21:04:49Z claimed by a-mh7sb8yq7e
- 2026-07-26T21:08:31Z adopted by a-root (owner a-nfazzjdrh2 orphaned)
- 2026-07-26T21:08:31Z accepted by a-root
- 2026-07-26T21:08:31Z completed by a-root
