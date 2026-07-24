---
id: d-filed-task-136-route-5-remaining-raw-join-inline-list-writers-in-internal-store
kind: note
note_kind: decision
created: 2026-07-24T09:14:55Z
created_by: a-fkza193f7w
about: [[084]]
---
# Filed task 136 (route 5 remaining raw-Join inline-list writers in internal/store through Front.SetList) as the single highest-value evidence-based change
## Chose
Filed task 136 (route 5 remaining raw-Join inline-list writers in internal/store through Front.SetList) as the single highest-value evidence-based change
## Rejected
Filing against the runtime sandbox_ro_args comma-split finding (f-runtime-list...) or re-scoping task 127's encoder fix
## Because
The runtime writers named in f-runtime-list were already routed through SetList by task 119, and 127 fixes the encoder (quoteListElem escaping) — neither touches these 5 sites. store.go:297, shortcutfiles.go:39/42, roles.go:40 still hand-roll '['+strings.Join(v,", ")+']' while their fields are read back via Front.GetList (store.go:238, roles.go:82-86, shortcutfiles.go:60-61), a confirmed write/read asymmetry that even survives 127. Finding f-3-more-unquoted-join-inline-list-writers explicitly recommends this exact follow-up and task 119 is the established precedent, so it is grounded in a reviewer finding + a real round-trip corruption, not speculation. I did NOT implement it (read-only grant; sandbox blocks go build/test anyway).
