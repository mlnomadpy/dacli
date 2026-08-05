---
id: f-298-complete-on-branch-dacli-298-window-now-scopes-decision-finding-issue
kind: note
note_kind: finding
created: 2026-08-04T20:44:56Z
created_by: a-maintainer-qtd48g
about: "[[298]]"
severity: moderate
---
# 298 complete on branch dacli/298-...: window now scopes decision + finding-issue mirrors, push prints a blast-radius plan line
Commit 2301322 by a-maintainer-qtd48g on branch dacli/298-the-github-push-task-window-scopes-tasks-but-decisions-ride-along-unscoped. Changed internal/features/ghmirror/ghmirror.go (+window scoping), window_test.go (+acceptance test), docs/GITHUB.md. Root cause: mirrorDecisions ran over ALL decisions unconditionally (ghmirror.go:403) and mirrorFindingIssues scoped only by --since, so a windowed 'github push core 275' still published every project decision — an unbounded public-repo disclosure. Fix: shared noteInWindow(doc, refTasks, since) predicate (about-match on ref-selected tasks OR created>=since, mirroring selectTaskWindow's two axes); mirrorDecisions/mirrorFindingIssues now take (notes, refTasks, since) and skip out-of-window notes; cmdPush prints 'plan: will create N task, M decision, K finding issue(s) on <repo>' before any create, counting only genuine creates via the in-memory idx snapshot. mirrorFindingsOnly now honors + validates refs instead of silently dropping them. VERIFIED: temporarily forced noteInWindow to return true (pre-change behavior) and re-ran TestPushWindowScopesDecisionsAndFindingsToTheWindow — it FAILED on both the leaked out-of-window decision and finding and the wrong plan counts (2 vs 1); reverted, test passes. go build/test ./... green, go vet clean, gofmt -l clean. Owner: accept + integrate --tasks 298.
