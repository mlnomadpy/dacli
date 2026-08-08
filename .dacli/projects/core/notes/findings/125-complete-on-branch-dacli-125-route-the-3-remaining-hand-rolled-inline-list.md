---
id: f-125-complete-on-branch-dacli-125-route-the-3-remaining-hand-rolled-inline-list
kind: note
note_kind: finding
created: 2026-07-24T09:11:40Z
created_by: a-qa5kgt0624
about: [[125]]
severity: moderate
---
# 125 complete on branch dacli/125-route-the-3-remaining-hand-rolled-inline-list-writers-through-the-quote-aware, all 3 acceptance criteria met
Committed a2e6cb7 by a-qa5kgt0624 (fixer). Staged only the 4 intended files: internal/store/store.go, internal/store/shortcutfiles.go, internal/store/roles.go, internal/store/inlinelist_test.go. (1) store.go:297 depends_on now calls d.Front.SetList("depends_on", opts.DependsOn) instead of hand-rolling '['+strings.Join(...)+']'. (2) shortcutfiles.go:39,42 params and roles both now call d.Front.SetList. (3) roles.go:40 the shared setList closure used for skills/scope/out_of_scope/shortcuts/escalate_to now calls d.Front.SetList instead of hand-rolling the join. All three route through mdstore.Front.SetList (internal/mdstore/mdstore.go:140), the quote-aware encoder task 119 landed. (4) New internal/store/inlinelist_test.go adds 3 round-trip tests mirroring TestRuntimeInlineListRoundTripsCommaContainingElements: TestTaskDependsOnRoundTripsCommaContainingElement (CreateTask -> mdstore.ReadFile), TestShortcutParamsAndRolesRoundTripCommaContainingElement (CreateShortcut -> LoadShortcut), TestRoleListFieldsRoundTripCommaContainingElement (CreateRole -> LoadRole) — each with a comma-containing element, all PASS. go build ./... clean; go test ./internal/... all green. Box-checking refused for non-owner (only a-k6yvk61byc) -- owner should verify and close via dacli task check/done + dacli merge --task 125.
