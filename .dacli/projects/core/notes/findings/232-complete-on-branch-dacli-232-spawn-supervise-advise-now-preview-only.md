---
id: f-232-complete-on-branch-dacli-232-spawn-supervise-advise-now-preview-only
kind: note
note_kind: finding
created: 2026-08-04T11:47:22Z
created_by: a-maintainer-vrmdqz
about: "[[232]]"
severity: moderate
---
# 232 complete on branch dacli/232-...: spawn/supervise --advise now preview-only, unified with loop --advise
Commit 9b63268 by a-maintainer-vrmdqz. Root cause: resolveLaunch (execution.go) printed the advisory then FELL THROUGH to gates+mint+exec, so 'spawn --advise' launched a real (billed) run, while 'loop --advise' returned without spawning — opposite meanings for one flag. Fix: resolveLaunch now returns sentinel errAdviseOnly right after printAdvisory (execution.go:~397), BEFORE any gate/agentid.Spawn/exec; cmdSpawn and cmdSupervise unwrap it to a clean exit-0 preview. printAdvisory's closing line changed to '(preview only — no agent spawned; re-run without --advise to launch)'. Now --advise means 'look, don't act' on spawn/supervise/loop alike, satisfying acceptance option 1 ('advise previews without acting on both commands'). Test: TestSpawnAdvisePreviewsWithoutSpawning (spawn_refusal_test.go) drives cmdSpawn with a runtime that CAN enforce ro + an installed binary, asserts exit 0 and ZERO agents/runs minted + the 'no agent spawned' readout — it FAILS before the change (delta of 1 minted identity) and passes after. Docs realigned: RUNTIMES.md §23 + flag table, SKILL.md warning, README.md, mcp_tools.md, git_workflow.md (+testdata golden). Proof: go build ./... clean; go vet ./... clean; gofmt -l internal/ empty; go test ./internal/features/execution ./internal/prompts ./internal/mcp green. NOTE: the only ./... failure is TestCatalogRefusesRatherThanWritingAnEmptyRoster, pre-existing (verified by re-running with my changes git-stashed) — the known DACLI_AGENT env-leak test-isolation gap in internal/features/catalog (no TestMain clearing the var), NOT this change. Owner: verify + close via dacli accept 232, then integrate/merge --task 232.
