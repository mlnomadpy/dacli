---
id: t-01KY9PFN43GM802T5DJSHYJETR
kind: task
created: 2026-07-24T09:14:43Z
created_by: a-fkza193f7w
owner: a-root
priority: should
github:
  issue: 232
  repo: mlnomadpy/dacli
---
# Route the 5 remaining raw-Join inline-list writers in internal/store through Front.SetList so list fields round-trip losslessly
## So that
role skills/scope/out_of_scope/shortcuts/escalate_to, shortcut params/roles, and task depends_on are WRITTEN with raw strings.Join (no quoting) but READ back via Front.GetList, breaking the documented SetList=inverse-of-GetList invariant (mdstore.go:136-146); an element with a top-level comma, quote, bracket, brace, '#', or leading/trailing space silently corrupts on read-back — the exact asymmetry task 119 fixed for runtimefiles.go/gates.go and finding f-3-more-unquoted-join-inline-list-writers flagged as an explicit follow-up
## Acceptance
- [x] internal/store/store.go:297 (depends_on) writes via d.Front.SetList("depends_on", opts.DependsOn) instead of "["+strings.Join(...)+"]"
- [x] internal/store/shortcutfiles.go:39,42 (params, roles) and internal/store/roles.go:38-41 setList helper (skills/scope/out_of_scope/shortcuts/escalate_to) all write via Front.SetList; no raw '['+strings.Join(...)+']' inline-list writer remains in internal/store (grep is clean)
- [x] a store round-trip test writes a role (or shortcut) whose list field holds an element containing a top-level comma AND a quote (e.g. it's "a,b"), reloads it, and asserts len==1 and byte-equality of that element
- [x] go build ./... clean and go test ./internal/store/... ./internal/mdstore/... green; no regression to any currently-passing case
## Log
- 2026-07-26T21:14:20Z claimed by a-wxkxsvvt3y
- 2026-07-26T21:16:30Z adopted by a-root (owner a-fkza193f7w orphaned)
- 2026-07-26T21:16:30Z accepted by a-root
- 2026-07-26T21:16:30Z completed by a-root
- 2026-08-03T22:38:15Z a-wxkxsvvt3y: PR opened: https://github.com/mlnomadpy/dacli/pull/264 (event 01KYG4JE8ZJ92M8RQQK32KHVFN)
