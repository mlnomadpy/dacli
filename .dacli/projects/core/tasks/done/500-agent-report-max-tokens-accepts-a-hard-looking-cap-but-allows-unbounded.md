---
id: t-01M0ZCAPM3YNJ2PJAJSZV4ATKX
kind: task
created: 2026-08-26T15:51:56Z
created_by: a-root
owner: a-root
github:
  issue: 796
  repo: mlnomadpy/dacli
estimate: "{optimistic: 5, probable: 8, pessimistic: 13}"
---
# [agent-report] max-tokens accepts a hard-looking cap but allows unbounded provisional runs
## Context
Adopted from GitHub issue #796.

Multiple runs launched with --max-tokens substantially exceeded the requested limit. Spawn warned that the cap was not enforced because the role/model band lacked sufficient measured history, then proceeded. This defeats operator cost governance: a parameter named max-tokens behaves as advisory precisely when calibration is missing. Expected: enforce the runtime/provider token ceiling when supported; otherwise fail closed before spawn unless the operator explicitly selects an advisory override. Acceptance criteria: an uncalibrated band cannot silently exceed --max-tokens; CLI output distinguishes enforced limits from estimates; tests cover unsupported runtime accounting and explicit override. Exact run measurements are intentionally withheld from the public issue but are available to maintainers through an approved private channel. Non-goal: requiring mature cost calibration merely to estimate a task.

---
_Reported via `dacli report`._
- dacli: dev
- platform: darwin/arm64
- workspace and run transcript withheld (public upstream) — re-run with --disclose to include them

## Acceptance
- [x] An uncalibrated role/model band cannot launch past `--max-tokens` unless the operator explicitly selects an advisory override.
- [x] A runtime/provider with a supported token ceiling receives and enforces that ceiling for the launched process.
- [x] CLI dry-run and launch output distinguish an enforced ceiling, a calibrated estimate, and an explicitly advisory limit.
- [x] Public-command tests cover supported enforcement, unsupported accounting refusal, and the explicit advisory override without requiring mature calibration merely to estimate work.
- [x] Mutation evidence and the full repository verification gates pass.
## Log
- 2026-08-26T22:35:37Z claimed by a-root
- 2026-08-26T23:12:04Z accepted by a-root
- 2026-08-26T23:12:04Z verified by `GOCACHE=/private/tmp/dacli-go-cache go test ./...` (exit 0) in branch main at d909be6 — proves that tree builds, not that the work is in trunk
- 2026-08-26T23:12:04Z deliverable: dacli/500-agent-report-max-tokens-accepts-a-hard-looking-cap-but-allows-unbounded is merged into main
- 2026-08-26T23:12:04Z completed by a-root
- 2026-08-26T23:13:14Z a-root: PR opened: https://github.com/mlnomadpy/dacli/pull/802 (event 01M1045REM2SP20NYP6DRQ2JH2)
## Verification Evidence
{"command":"GOCACHE=/private/tmp/dacli-go-cache go test ./...","exit_code":0,"duration_ms":37822,"artifact_hash":"sha256:b2b559580bfabdcbbc6cfb16531aec747959461a63d9dca39404dd76f9ae0759","verifier":"a-root","branch":"dacli/500-agent-report-max-tokens-accepts-a-hard-looking-cap-but-allows-unbounded","commit_sha":"c0f01cb81978cd3eb29a43ef8c5101324a179570"}
{"command":"GOCACHE=/private/tmp/dacli-go-cache go test ./...","exit_code":0,"duration_ms":69161,"artifact_hash":"sha256:0f85476a17371ca16da296c872373557d2f8c8d50cfed8611b0718195db8c8d9","verifier":"a-root","branch":"main","commit_sha":"d909be6ce6828db9da41d921c9839d589cf4d466"}
