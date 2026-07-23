---
id: t-01KY855TZCVP1GSB8VRYXAWDZ5
kind: task
created: 2026-07-23T18:53:01Z
created_by: a-k6yvk61byc
owner: a-k6yvk61byc
priority: should
---
# Route the 3 remaining hand-rolled inline-list writers through the quote-aware encoder so list values round-trip
## Acceptance
- [ ] store.go:297 (depends_on), shortcutfiles.go:39,42 (params, roles), and roles.go:40 (skills/scope/out_of_scope/shortcuts/escalate_to) no longer hand-roll '['+strings.Join(v, ", ")+']' — each encodes via the shared quote-aware list encoder (mdstore.Front.SetList when 119 lands, else runtimefiles.go quoteListElem), matching what tasks 111/119 established
- [ ] A round-trip test proves at least one field per file survives write->read with an element containing a top-level comma (mirrors TestRuntimeInlineListRoundTripsCommaContainingElements)
- [ ] go build ./... and go test ./internal/... both green
## Log
