---
id: f-076-complete-on-branch-dacli-076-prompt-upgrade-all-prompt-templates-reviewed
kind: note
note_kind: finding
created: 2026-07-22T19:58:24Z
created_by: a-nffxy9dhbg
about: [[076]]
severity: moderate
---
# 076 complete on branch dacli/076-prompt-upgrade — all prompt templates reviewed + upgraded to current surface
Commit fc553fe by a-nffxy9dhbg. Files: internal/prompts/tpl/{git_workflow.md,mcp_tools.md,protocol_preamble.md} + docs/PROMPTS.md (registry table rows). Upgrades: (1) git_workflow.md replaces the stale 'owner merges with dacli merge' line with the current owner close-out (accept verifies+box-checks+done in one step, then integrate --tasks/--into or ship for a wave, merge for one branch) and adds a decompose-and-delegate paragraph referencing spawn --detach/--claim/--advise/--max-tokens, wait, agents --tail, and the taint/token-calibration spawn gates. (2) mcp_tools.md 'cli' escape-hatch section now maps the full command surface: spawn/wait/agents(--tail)/logs/kill lifecycle, accept/integrate/ship/merge/commit/push/pr close-out, calibrate/estimate/taint gates, github push/pull/sync. (3) protocol_preamble.md finding path now states findings enter sibling briefs tagged 'unverified' until an adversarial dacli verify panel confirms/refutes (trust gate). All referenced command names verified present in the command table (execution.go/ship.go/acceptance.go/vcs/lifecycle.go/ghmirror.go); no stale names. review_workflow/supervise_correction/brief_header/refusal_next/verify_refute reviewed, confirmed accurate to shipped behavior, unchanged. mcp_tools still documents exactly the 16 registered MCP tools (mcp/tools.go). go build ./... clean; go test ./internal/... all green incl. internal/prompts (TestRenderPreamble, TestMCPDescSections) and internal/cli (TestTiering git_workflow assertions, spawn/supervise). Owner: verify + close via dacli accept 076, then integrate/merge --task 076.
