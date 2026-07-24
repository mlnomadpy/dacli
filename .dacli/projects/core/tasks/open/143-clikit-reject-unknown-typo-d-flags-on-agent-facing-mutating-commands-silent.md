---
id: t-01KY9Q6TVH2GC0MAJC2XQE7W6B
kind: task
created: 2026-07-24T09:27:22Z
created_by: a-vh51d10ng9
owner: a-vh51d10ng9
priority: should
---
# clikit: reject unknown/typo'd flags on agent-facing mutating commands (silent flag-drop loses provenance)
## So that
a typo'd or unsupported flag fails loudly (exit 2) instead of silently dropping the agent's data with exit 0
## Context
ROOT CAUSE: clikit.ParseFlags (internal/clikit/clikit.go:104-142) collects every --key into an internal map with no allowlist; each command reads only the keys it knows via f.Get, so any other flag is silently ignored (exit 0). EVIDENCE: (1) cmdTaskAdd (internal/features/planning/planning.go:133-140) reads only project/force/priority/estimate/accept/so-that/context/depends-on/parent -- 'dacli task add x --project core --body ...' drops --body, and a typo like --acccept drops the acceptance criterion, with NO error. (2) Prior finding f-dacli-init-silently-ignores-template-and-roster: dacli init silently ignored two spec-advertised flags; it was patched to hand-validate ITS flags (wscore.go:30-45), but a typo (--tempate) there still drops silently -- proving the per-command patch does not fix the class. (3) clikit.go:96-97 cites run 01KY2K8N4C corruption from silent flag defaulting. IMPACT: in an autonomous swarm where 'work not reported does not exist', a dropped --because on a decision or --accept on a task add loses provenance invisibly. DESIGN CONSTRAINT: Flags.Raw() exists BECAUSE 'run' forwards unknown flags (clikit.go:153-155), so a global reject in ParseFlags is WRONG -- the fix must be an OPT-IN per-command strict check, not applied to pass-through commands. Do NOT implement here; this task files the work.
## Acceptance
- [ ] Add an opt-in strict-flag helper to clikit.Flags (e.g. Reject/Unknown(known ...string) error) that returns Usagef (exit 2) naming the offending flag(s); a clikit unit test proves an unknown flag is rejected and a correct known-set passes
- [ ] Adopt the helper in the agent-facing mutating commands where a dropped flag loses work -- at minimum 'task add', 'note add', and 'task check' -- while leaving Raw()-forwarding commands (run) untouched
- [ ] Regression test: 'dacli task add <t> --project core --acccept y' (typo) exits 2 with an actionable message, and the same command with correct --accept still succeeds; go build ./... and go test ./... are green and gofmt -l . is clean
## Log
