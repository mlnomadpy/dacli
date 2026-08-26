---
id: t-01M0CZANEM3TFEMGTW3NTNXGXM
kind: task
created: 2026-08-19T12:18:23Z
created_by: a-root
owner: a-root
github:
  issue: 715
  repo: mlnomadpy/dacli
estimate: "{optimistic: 3, probable: 5, pessimistic: 8}"
depends_on: "[t-01M0F3795JGCAG6ZS3XVAGNS2J, t-01M0F8JAH5CNJ327M31B1821BF]"
---
# [agent-report] runtime doctor marks unauthenticated Claude runtime usable
## Context
Adopted from GitHub issue #715.

dacli runtime doctor reported claude-ro available, then task 011 security-reviewer run 01M0CYBC6K exited immediately with 'Not logged in · Please run /login', yielding no visible result. Expected: runtime doctor/pre-spawn validation detects missing authentication and refuses before spending a governed run.

---
_Reported via `dacli report`._
- dacli: dev
- platform: darwin/arm64
- workspace and run transcript withheld (public upstream) — re-run with --disclose to include them

## Acceptance
- [x] Runtime health distinguishes an installed CLI binary from an authenticated, usable runtime.
- [x] The Claude adapter detects the reproducible `Not logged in` state and reports an actionable authentication remedy.
- [x] `preflight` or `spawn` refuses before creating/spending a governed agent run when required runtime authentication is unavailable.
- [x] Authentication probes are adapter capabilities so Codex, Claude Code, Gemini, Copilot, and generic runtimes can implement provider-specific checks without provider logic in the scheduler.
- [x] Tests simulate a version-capable but unauthenticated runtime and prove doctor, preflight, and spawn agree on usability.
- [x] Mutation proof demonstrates bypassing the authentication result makes the focused test fail.
## Log
- 2026-08-20T09:08:25Z dependency edit by a-root (event 01M0F6VH5A90WKE1R4NMV7PQG4)
- 2026-08-20T09:38:28Z dependency edit by a-root (event 01M0F8JHXXZW2DQWKSKHZ1FRAM)
- 2026-08-22T21:55:20Z claimed by a-fixer-76xc6t
- 2026-08-26T12:45:02Z completed by a-root
- 2026-08-26T12:55:11Z a-root: PR opened: https://github.com/mlnomadpy/dacli/pull/776 (event 01M0Z18N9PWPEZG7T1T6QM97X9)
## Verification Evidence
{"command":"GOCACHE=/private/tmp/dacli-go-cache go test ./internal/features/execution -run 'TestUnauthenticatedClaudeIsRefusedBeforeSpawn|TestAuthenticatedClaudeInitIsLaunchCompatible|TestLegacyClaudeInferenceRejectsUnknownExecutionFlag' -count=1","exit_code":0,"duration_ms":1511,"artifact_hash":"sha256:ebef43b7bdbb613fbacde20f7cffd6da65dc9669365736429581029ef3ac3e61","verifier":"a-root","branch":"dacli/480-agent-report-runtime-doctor-marks-unauthenticated-claude-runtime-usable","commit_sha":"a38a2192810acbf78eb13cd860b460c347e54955"}
