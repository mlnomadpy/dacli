---
id: f-119-complete-mdstore-front-setlist-added-both-known-call-sites-routed-through
kind: note
note_kind: finding
created: 2026-07-23T18:50:08Z
created_by: a-bk3fqg83sm
about: [[119]]
severity: moderate
---
# 119 complete: mdstore Front.SetList added, both known call sites routed through it, round-trip test added, build+tests green
internal/mdstore/mdstore.go: new Front.SetList(k, []string) (SetList sits right after GetList) encodes an inline list via the moved-in quoteListElem helper (same quoting rules as before: quote on comma/bracket/brace/#/quote chars or leading/trailing whitespace), then Set()s the bracketed, comma-joined string -- exact inverse of GetList/splitTop/clean. internal/store/runtimefiles.go setInline (was runtimefiles.go:79-83) now calls d.Front.SetList(k, v) instead of hand-rolling the join+quote; its local quoteListElem duplicate was deleted. internal/gates/gates.go writePhase (gates.go:313) now calls p.Doc.Front.SetList("phase_allows", s.Allow) instead of hand-rolling strings.Join; the len(s.Allow)==0 branch still Deletes the key (unchanged empty-list semantics, SetList itself is unconditional like Set/SetBlock). New TestSetListRoundTrip in internal/mdstore/mdstore_test.go proves SetList->GetList identity for elements with commas, spaces, brackets, braces, single/double quotes, and empty/nil lists. go build ./... clean; go test ./internal/... all green. gofmt -l . clean. Filed a separate finding about 3 more unquoted-join sites (store.go/shortcutfiles.go/roles.go) found while grepping for duplicates but left out of scope.
