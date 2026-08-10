---
id: f-316-complete-on-branch-dacli-316-three-code-cited-architecture-diagrams
kind: note
note_kind: finding
created: 2026-08-10T13:54:58Z
created_by: a-maintainer-a87zyw
about: "[[316]]"
severity: major
---
# 316 complete on branch dacli/316: three code-cited architecture diagrams committed
Commit ac19da3 (a-maintainer-a87zyw). All 3 acceptance criteria met.

(1) Diagrams committed as text so they diff: docs/DIAGRAMS.md holds Mermaid fenced blocks (mkdocs already renders these via pymdownx.superfences mermaid, mkdocs.yml:44-48). Linked from nav (mkdocs.yml), docs/README.md, and ARCHITECTURE.md 2b.

(2) The set covers all three required views: a COMPONENT graph (graph TD — the 4 FSD layers, all 21 feature slices, entity + shared packages, enforced import rules), a SEQUENCE for spawn->landing (sequenceDiagram — gates, worktree, runtime, child commit/event crumbs, wait/sync/close, --pr vs --no-pr land), plus a loop-cycle graph, and a task-lifecycle STATE machine (stateDiagram-v2 — open/active/blocked/done, owner vs propose paths, the pendingAccept hold).

(3) Each diagram is checked against code: every edge has a file:line citation in an 'Edges -> code' table beneath its diagram (e.g. launchGates execution.go:371-378, cli aggregate cli.go:24-83, CloseTask store.go:1249, reconcilePendingAccepts orchestration.go:897-931, sync apply sync.go:179-234). Verified by three parallel code-reading passes cross-checked against my own reads of cli.go, execution.go, orchestration.go, planning.go, model.go, arch_test.go.

Anti-drift test (fails before change): internal/cli/diagrams_test.go — TestDiagramsCoverEveryFeatureSlice fails the build if any internal/features/* production slice is not named in DIAGRAMS.md; TestDiagramsHaveAllThreeViews asserts the mermaid/graph/sequence/state blocks exist. Red-green verified: removed 'skillforge' from the doc -> FAIL naming skillforge; restored -> ok.

Also filed: ARCHITECTURE.md 2b slice/entity lists are stale (names 10 slices, code has 21; entity layer omits gates/gitx/procmon/agentstate/skills) — DIAGRAMS.md is now the current inventory and 2b gained a forward-link + a note that its lists lag.

PROOF: go build ./... clean, go vet ./... clean, gofmt -l internal/ clean. go test ./... green under the CI convention (go test -exec 'env -u DACLI_AGENT' ./...); the only 3 reds under plain 'go test' are briefing/catchup_test.go hitting this dogfood session's ambient DACLI_AGENT (task 288 fail-closed), unrelated to this docs-only change and green with the env unset.

Owner: dacli accept 316 (task check is gated to a-root; I could not check the boxes myself). PR-first is off — branch dacli/316-redraw-the-architecture-diagrams-to-match-the-system-that-exists-now is ready for accept + integrate/merge --task 316.
