---
id: f-291-complete-on-branch-dacli-291-central-json-gate-invariant-test-build-test
kind: note
note_kind: finding
created: 2026-08-04T20:45:49Z
created_by: a-maintainer-7zg8zg
about: "[[291]]"
severity: moderate
---
# 291 complete on branch dacli/291-...: central --json gate + invariant test, build/test/vet/gofmt clean
Commit c01042c by a-maintainer-7zg8zg. clikit.Command gains a JSON bool; cli.go adds refuseUnsupportedJSON wired into BOTH Main (cli.go:131) and the MCP executor (cli.go:231) so --json is refused (exit 2, usage) for any command whose Command.JSON is false, with a hint listing the honoring commands built from the table (jsonCommands, indirected via jsonCmdList to break the init cycle). Marked JSON:true: context, task list (emit JSON docs), init, new (adapt output). TestJSONFlagIsHonoredOrRefused (internal/cli/json_invariant_test.go) drives all 116 commands through the real executor with --json and asserts every non-honoring command refuses exit-2; a pinned jsonHonoringCommands map fails the test if a new command sets Command.JSON without being recorded. TestJSONHonoringCommandsEmitOrAdapt validates context/task list emit valid JSON and init suppresses decoration. go build/test ./... green, go vet clean, gofmt -l empty. Verified fail-before-fix by stubbing the gate to return nil: the invariant then fails for whoami/project list/task add/etc (code!=2). PR-first off: owner accepts + integrates.
