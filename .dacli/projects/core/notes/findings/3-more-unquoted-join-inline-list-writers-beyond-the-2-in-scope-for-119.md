---
id: f-3-more-unquoted-join-inline-list-writers-beyond-the-2-in-scope-for-119
kind: note
note_kind: finding
created: 2026-07-23T18:49:53Z
created_by: a-bk3fqg83sm
about: [[119]]
severity: minor
---
# 3 more unquoted-join inline-list writers beyond the 2 in scope for 119
internal/store/store.go:297 (depends_on), internal/store/shortcutfiles.go:39,42 (params, roles), internal/store/roles.go:40 (generic setInline-style writer) all still hand-roll '['+strings.Join(v, ", ")+']' instead of the new mdstore Front.SetList (internal/mdstore/mdstore.go). Same antipattern as runtimefiles.go/gates.go had: an element containing a comma (e.g. a depends_on task ref with a comma in it, unlikely but not validated against) would mis-split on GetList. Left untouched because task 119's acceptance named only the runtimefiles.go/gates.go pair as in scope; a follow-up task should route these through Front.SetList too.
