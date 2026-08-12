---
id: f-five-test-packages-still-leak-the-dogfood-agent-token-into-temporary-workspaces
kind: note
note_kind: finding
created: 2026-08-12T14:18:59Z
created_by: a-codex-loop-auditor-g1st46
about: "[[376]]"
severity: major
---
# Five test packages still leak the dogfood agent token into temporary workspaces
Reproduced from this live dacli agent with GOCACHE=/private/tmp/dacli-376-gocache go test ./...: internal/features/briefing catchup_test.go:29/55/76, internal/features/orchestration noremote_test.go:187, and internal/features/teamops teamops_test.go:653 fail with 'agent token not recognized in this workspace'; internal/cli's E2E child also fails for the same ambient-token class. The command tests create fresh temporary workspaces but inherit this session's DACLI_AGENT, whose identity exists only in the real workspace, so identity resolution fails before the asserted behavior. Task 262 fixed only catalog; task 288's completion note explicitly claimed plain go test was green after migrating the empty-token sites, but these current non-empty foreign-token failures remain. Running commands with DACLI_AGENT unset is the manual workaround, but CONTRIBUTING.md requires plain go test ./....
