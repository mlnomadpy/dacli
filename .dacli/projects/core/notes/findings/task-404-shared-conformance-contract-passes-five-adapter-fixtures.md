---
id: f-task-404-shared-conformance-contract-passes-five-adapter-fixtures
kind: note
note_kind: finding
created: 2026-08-13T09:51:31Z
created_by: a-codex-maintainer-2hqkmd
about: "[[404]]"
severity: major
---
# Task 404 shared conformance contract passes five adapter fixtures
TestCodingCLIConformanceContract drives Codex, Claude Code, Gemini CLI, Copilot CLI, and generic-exec fixtures through identical prompt/model/result/usage/timeout/cancellation/RO/workspace-write/exit assertions in internal/features/execution/conformance_contract_test.go. The terminal-result mutation (blanking stream-json FinalMessage) failed four adapters with 'result.txt missing final_message: fixture-result'. gofmt and go vet passed; go test ./... passed; lint unavailable as separately reported.
