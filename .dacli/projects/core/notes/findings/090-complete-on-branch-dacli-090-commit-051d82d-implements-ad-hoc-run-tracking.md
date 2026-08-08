---
id: f-090-complete-on-branch-dacli-090-commit-051d82d-implements-ad-hoc-run-tracking
kind: note
note_kind: finding
created: 2026-07-23T10:08:55Z
created_by: a-e3jt52gb4k
about: [[090]]
severity: moderate
---
# 090 complete on branch dacli/090: commit 051d82d implements ad-hoc run tracking + shortcut promote
Committed 051d82d by a-e3jt52gb4k (fixer) via git add + dacli commit --no-add, staging exactly the 7 intended files: docs/ARCHITECTURE.md, docs/SHORTCUTS.md, internal/cli/{agents_run_test.go,cli.go,supervise_test.go}, internal/features/governance/governance.go (deleted), internal/features/shortcuts/shortcuts.go. Both acceptance items satisfied: (1) internal/features/shortcuts/shortcuts.go: 'dacli run --cmd <literal>' executes an ad-hoc command not backed by a shortcut file and records it as an attributed EventRun, About = adhocKey(cmd) = 'adhoc:' + sha256(cmd)[:12] (a wikilink-safe content hash, since About is a [[...]] wikilink target and raw command text is unconstrained); read-only agents are refused (exit 3) since an ad-hoc command has no declared effect to gate on finer than write. 'dacli shortcut promote <name> --from-event <id> --effect ...' resolves the run event by ULID, refuses (exit 3) a target whose About is not adhoc-prefixed (already a named shortcut) or whose literal command has run fewer than 2 times, then calls store.CreateShortcut with the event's captured command text. (2) internal/features/governance/governance.go deleted entirely (it held only the one clikit.Planned stub); shortcut promote now lives in the shortcuts slice's Commands table; cli.go's governance import/registration removed. Covered by new TestShortcutPromote (internal/cli/agents_run_test.go): ro-refused, dry-run, single-run refusal, repeated-run promotion, and the promoted shortcut running. go build ./... clean, gofmt -l . clean, go vet ./... clean, go test ./... all green including TestFeatureSlicesAreIsolated and TestAppLayerStaysThin. docs/SHORTCUTS.md and docs/ARCHITECTURE.md updated to describe the shipped behavior instead of the stub. Owner: verify and close via dacli task check/done + dacli merge --task 090.
