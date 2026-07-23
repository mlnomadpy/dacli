---
id: t-01KY64CA5PC2Z8MD854G5ZMNW8
kind: task
created: 2026-07-23T00:00:36Z
created_by: a-48ab0df8g5
owner: a-root
priority: should
---
# Fix onboard TODO-scanner: markers matched as bare substrings in string literals and comments pollute the codebase map and --todos seeding
## So that
every brief carries an honest 'Open markers' list and adopt --todos seeds real work, not the scanner matching its own source
## Acceptance
- [x] scanTodos (internal/features/onboard/onboard.go:243) reports a marker only when it appears as a standalone token in a comment, not as an arbitrary substring — Go string literals containing marker names (onboard.go:249 []string{"TODO","FIXME","HACK","XXX"}; gates.go:465 []string{"TBD","TODO","FIXME",...}; the command Brief onboard.go:30 "...codebase map, TODO tasks...") and test-fixture assertion strings (onboard_test.go:52,55) no longer produce entries
- [x] a regression test walks a fixture containing (a) a real // TODO: handle x comment, (b) a Go line s := "TODO in a string", and (c) []string{"TODO","FIXME"} and asserts exactly one marker is reported (the real comment), with its file:line and text
- [x] running dacli on this repo's own tree yields a codebase-map 'Open markers' list free of self-referential matches from onboard.go and gates.go source; the existing TestAdoptExistingRepo still passes (the intended pay.go // TODO: handle the batch path marker is still found and, under --todos, seeded)
- [x] committed by an agent; go build ./... and go test ./internal/... green
## Log
- 2026-07-23T00:01:18Z claimed by a-bndgc6d73j
- 2026-07-23T00:14:20Z adopted by a-root (owner a-48ab0df8g5 orphaned)
- 2026-07-23T00:14:20Z accepted by a-root
- 2026-07-23T00:14:20Z completed by a-root
