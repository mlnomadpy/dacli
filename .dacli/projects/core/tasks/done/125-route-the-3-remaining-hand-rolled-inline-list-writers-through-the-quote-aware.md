---
id: t-01KY855TZCVP1GSB8VRYXAWDZ5
kind: task
created: 2026-07-23T18:53:01Z
created_by: a-k6yvk61byc
owner: a-root
priority: should
---
# Route the 3 remaining hand-rolled inline-list writers through the quote-aware encoder so list values round-trip
## Acceptance
- [x] store.go:297 (depends_on), shortcutfiles.go:39,42 (params, roles), and roles.go:40 (skills/scope/out_of_scope/shortcuts/escalate_to) no longer hand-roll '['+strings.Join(v, ", ")+']' — each encodes via the shared quote-aware list encoder (mdstore.Front.SetList when 119 lands, else runtimefiles.go quoteListElem), matching what tasks 111/119 established
- [x] A round-trip test proves at least one field per file survives write->read with an element containing a top-level comma (mirrors TestRuntimeInlineListRoundTripsCommaContainingElements)
- [x] go build ./... and go test ./internal/... both green
## Log
- 2026-07-24T09:08:56Z claimed by a-qa5kgt0624
- 2026-07-24T09:12:12Z adopted by a-root (owner a-k6yvk61byc orphaned)
- 2026-07-24T09:12:12Z accepted by a-root
- 2026-07-24T09:12:12Z completed by a-root
