---
id: t-01M0F3795JGCAG6ZS3XVAGNS2J
kind: task
created: 2026-08-20T08:04:56Z
created_by: a-root
owner: a-root
github:
  issue: 746
  repo: mlnomadpy/dacli
estimate: "{optimistic: 5, probable: 8, pessimistic: 13}"
---
# Add launch-compatible behavioral preflight for coding-agent runtimes
## Context
Adopted from GitHub issue #746.

## Reproduction

On macOS on 2026-08-19, `runtime doctor` marked `codex-ro` usable and
`dacli spawn` accepted audit tasks 011 and 013. Runs `01M0D6KC42` and
`01M0D6KC7Z` then failed before reasoning with:

```text
failed to initialize in-process app-server client: Operation not permitted (os error 1)
```

`dacli agents` showed no live child and `dacli wait` had nothing to finalize.
The binary/version probe was true, but the exact adapter invocation was not
launch-compatible in the effective sandbox.

## Design direction

Give every runtime adapter an optional bounded behavioral-preflight strategy in
addition to binary/version and authentication probes. Probe the same preset,
grant, sandbox, user-configuration policy, and startup transport that `spawn`
will use, but with a no-work handshake and a short deadline. Return structured
capabilities and categorized failures (`authentication`, `sandbox`, `startup`,
`quota`, `transport`) rather than parsing provider prose in scheduling code.

`spawn` must consume a fresh compatible result or run the bounded probe before
minting a child, claiming a task, or creating a worktree. A deterministic
incompatibility is policy refusal (exit 3) with the failing layer and an
actionable alternative; transient probe failures remain retryable errors. Keep
provider-specific argv and response parsing inside adapters so Codex, Claude
Code, Gemini, Copilot, and generic runtimes share one orchestration contract.

This task covers startup/sandbox/transport compatibility. Task 480 covers the
separate authentication-readiness signal; both should use the same structured
preflight result without collapsing their regressions into one test.

---
_Reported via `dacli report`._
- dacli: dev
- platform: darwin/arm64
- workspace and run transcript withheld (public upstream) — re-run with --disclose to include them

## Acceptance
- [ ] Runtime doctor reports binary presence/version separately from behavioral launch compatibility for the exact runtime preset and grant.
- [ ] The preflight contract represents authentication, sandbox, startup, quota, and transport outcomes without provider-specific branches in spawn scheduling.
- [ ] A Codex `grant=ro` fixture reproducing the app-server initialization refusal fails at preflight; child, task-claim, launched-run, and worktree records remain unchanged.
- [ ] The refusal uses exit 3, names the incompatible layer, and reports a configured compatible runtime or an explicit remediation command when one exists.
- [ ] A bounded timeout and cache freshness rule prevent a hung or expired preflight result from authorizing the same spawn invocation.
- [ ] Codex read-write and the existing Claude Code, Gemini, Copilot, and generic adapter paths retain their declared grant and sandbox behavior.
- [ ] Human and JSON doctor/preflight output expose probed-versus-declared provenance and the command timestamp without leaking credentials or prompts.
- [ ] Mutation evidence shows the Codex incompatibility fixture fails when spawn stops consulting behavioral compatibility; focused runtime/execution tests and `go test ./...` pass.
## Log
- 2026-08-20T09:08:50Z claimed by a-maintainer-dxj9ch
