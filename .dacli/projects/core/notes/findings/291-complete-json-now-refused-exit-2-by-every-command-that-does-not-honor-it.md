---
id: f-291-complete-json-now-refused-exit-2-by-every-command-that-does-not-honor-it
kind: note
note_kind: finding
created: 2026-08-05T13:38:04Z
created_by: a-fixer-fqabnj
about: "[[291]]"
severity: minor
---
# 291 complete: --json now refused (exit 2) by every command that does not honor it, gated centrally in cli.go
Found the branch already carried a complete implementation (commit c01042c, authored by a prior agent a-maintainer-7zg8zg) when I claimed 291: internal/clikit.Command gained a JSON bool field (clikit.go:34-46); the app-layer dispatch (internal/cli/cli.go:131-181, both Main and the MCP executor) now calls refuseUnsupportedJSON before Run, returning a usage error (exit 2, clikit.Usagef) naming the commands that do honor --json for any command with JSON=false. Only context, task list, init, and new set JSON:true (briefing.go, planning.go, wscore.go, onboard.go) -- context and task list emit real JSON documents, init/new adapt by dropping human-only decoration. internal/cli/json_invariant_test.go TestJSONFlagIsHonoredOrRefused drives every one of the 117 commands in the aggregated table through the real executor with json=true and asserts refusal (exit 2 + does-not-support-json message) unless JSON:true, plus cross-checks the declared set against a recorded jsonHonoringCommands map so a new command cannot silently start accepting the flag. Both acceptance criteria are met by this design: (1) every command refuses or honors --json rather than silently dropping it, (2) the invariant test enumerates the full table. Verified clean: go build ./..., go vet ./..., gofmt -l . (empty), go test ./... all green (internal/cli 9.75s incl the invariant test). I did not write new code for this task beyond verification -- pushed the existing branch and opened https://github.com/mlnomadpy/dacli/pull/374 with auto-merge queued. I could not check the task acceptance boxes myself (task check 291 refused: only the owner a-root checks acceptance boxes) -- leaving that, and accept/ship, to a-root.
