---
id: f-dashboard-frontend-tests-unverified-in-this-sandbox-npm-install-requires
kind: note
note_kind: finding
created: 2026-08-05T13:52:27Z
created_by: a-fixer-015xkz
about: "[[270]]"
severity: minor
---
# Dashboard frontend tests unverified in this sandbox: npm install requires network approval headless cannot grant
internal/features/dashboard/ui/src/components/AgentRow.vue, types.ts, and the AgentRow.test.ts spec were extended to cover the two new agent states (blocked, silent) added by this task's shared agentstate.Derive, since the dashboard API can now emit them. node_modules was not installed in this worktree and npm install returned This command requires approval with no interactive approver available headless, so vitest could not be run to confirm AgentRow.test.ts still passes. The edits are minimal and mirror the existing stalled case exactly (two new array entries in AGENT_STATES, two new switch cases each copying the stalled branch's shape) so risk is low, but they are unverified by the actual test runner. The Go side (agentstate, dashboard.go, execution.go) is fully covered: go build, go vet, and go test ./... all green.
