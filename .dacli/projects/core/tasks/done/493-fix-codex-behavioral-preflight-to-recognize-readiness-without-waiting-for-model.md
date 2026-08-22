---
id: t-01M0MZ05DNTWBYQSTYP5X8J2NF
kind: task
created: 2026-08-22T14:46:35Z
created_by: a-root
owner: a-root
estimate: "{optimistic: 5, probable: 8, pessimistic: 13}"
github:
  issue: 767
  repo: mlnomadpy/dacli
---
# Fix Codex behavioral preflight to recognize readiness without waiting for model completion
## Acceptance
- [x] A Codex JSONL probe becomes compatible after the first valid provider lifecycle readiness event rather than waiting for the complete model turn.
- [x] The probe terminates and reaps its child process after readiness, with no surviving process or goroutine on success, timeout, malformed output, or cancellation.
- [x] A startup that takes longer than five seconds but emits readiness within the documented bounded deadline is not classified as a transport failure.
- [x] Authentication, sandbox, quota, malformed-stream, early-exit, and true-timeout failures remain fail-closed with provider-neutral state and layer classifications.
- [x] The readiness deadline and lifecycle event contract are adapter-owned or explicitly versioned; cache keys prevent evidence from an older strategy authorizing a changed probe.
- [x] Tests cover fragmented JSONL, stderr noise, capped output, readiness followed by a hanging process, timeout before readiness, and mutation evidence for the readiness predicate.
- [x] A fresh-cache runtime doctor against the bundled Codex CLI reports codex-rw compatible, and a bounded dacli loop can pass preflight without a manual cache edit.
- [x] gofmt -l ., go vet ./..., the pinned golangci-lint run, and go test ./... pass.
## Log
- 2026-08-22T15:19:01Z accepted by a-root
- 2026-08-22T15:19:01Z verified by `GOCACHE=/tmp/dacli-go-cache-493-trunk go test ./internal/features/execution ./internal/store -run 'TestCodexBehavioralPreflight|TestLegacyCodex|TestBootstrapCodex|TestRuntimeDoctorJSONExposesInferred|TestLaunchPreflightCacheIncludes|TestRuntimePreset' -count=1` (exit 0) in branch main at c914fe8 — proves that tree builds, not that the work is in trunk
- 2026-08-22T15:19:01Z deliverable: dacli/493-fix-codex-behavioral-preflight-to-recognize-readiness-without-waiting-for-model is merged into main
- 2026-08-22T15:19:01Z completed by a-root
- 2026-08-22T15:35:36Z a-root: PR opened: https://github.com/mlnomadpy/dacli/pull/769 (event 01M0N042PCZ9WTCC0SJWDC324Y)
- 2026-08-22T15:35:36Z a-root: Landing policy override: mode=pr base=main (event 01M0N0RKPTX4WE6TMFKYS1K8W2)
- 2026-08-22T15:35:36Z a-root: Integrated via PR https://github.com/mlnomadpy/dacli/pull/769 at merge commit c914fe8dcdef38ec744b6162023f4a83b6227577 into main (event 01M0N0RXC7FV0A11X43QNEN4Z6)
## Verification Evidence
{"command":"GOCACHE=/tmp/dacli-go-cache-493-trunk go test ./internal/features/execution ./internal/store -run 'TestCodexBehavioralPreflight|TestLegacyCodex|TestBootstrapCodex|TestRuntimeDoctorJSONExposesInferred|TestLaunchPreflightCacheIncludes|TestRuntimePreset' -count=1","exit_code":0,"duration_ms":3711,"artifact_hash":"sha256:9424665c90e1da96a655a4f25e0afbce1b5c256afef893854b5d7f1499e74fae","verifier":"a-root","branch":"main","commit_sha":"c914fe8dcdef38ec744b6162023f4a83b6227577"}
{"command":"GOCACHE=/tmp/dacli-go-cache-493-trunk go test ./internal/features/execution ./internal/store -run 'TestCodexBehavioralPreflight|TestLegacyCodex|TestBootstrapCodex|TestRuntimeDoctorJSONExposesInferred|TestLaunchPreflightCacheIncludes|TestRuntimePreset' -count=1","exit_code":0,"duration_ms":3076,"artifact_hash":"sha256:f2362b1e5bc9220777da0889013f6a1ae352e1812af40d0b12b984d5c51fe9ed","verifier":"a-root","branch":"main","commit_sha":"c914fe8dcdef38ec744b6162023f4a83b6227577"}
