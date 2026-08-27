---
id: t-01M11RCG32CHW4VTAJSPZBTK6F
kind: task
created: 2026-08-27T14:01:06Z
created_by: a-root
owner: a-root
priority: must
estimate: "{optimistic: 5, probable: 8, pessimistic: 13}"
github:
  issue: 833
  repo: mlnomadpy/dacli
---
# Add explicit single-harness and hybrid routing policy
## So that
An agent-first loop stays on the operator-selected coding harness across implementation, review, recovery, and fallback unless cross-harness execution was explicitly authorized.
## Acceptance
- [x] Runtime documents persist an opaque harness family shared by read-only and read-write adapters; all shipped presets declare the correct family without scheduler vendor-name branches.
- [x] Operating profiles persist a harness policy with single as the safe default and hybrid as an explicit mode with an allowlist; profile preview and loop command output show the resolved policy.
- [x] Automatic implementation routing, review-role selection, and runtime fallback exclude roles outside the resolved harness policy; explicit incompatible roles are refused before any child is spawned with a concrete remediation.
- [x] A Codex single-harness integration fixture proves Codex implementation and Codex review are selected even when cheaper or preferred Claude roles exist; a hybrid fixture proves explicitly allowed cross-harness routing remains possible.
- [x] CLI, operator documentation, and the dacli skill explain harness selection separately from model selection and document single-harness and hybrid workflows.
- [x] The focused tests are mutation-proven and gofmt -l ., go vet ./..., golangci-lint run, and go test ./... pass.
## Log
- 2026-08-27T14:01:57Z claimed by a-root
- 2026-08-27T15:24:43Z accepted by a-root
- 2026-08-27T15:24:43Z verified by `go test ./...` (exit 0) in branch main at ee01a9a5 — proves that tree builds, not that the work is in trunk
- 2026-08-27T15:24:43Z deliverable: dacli/524-add-explicit-single-harness-and-hybrid-routing-policy is merged into main
- 2026-08-27T15:24:43Z completed by a-root
- 2026-08-27T21:45:52Z a-root: PR opened: https://github.com/mlnomadpy/dacli/pull/834 (event 01M11SF84Z9644S7S9F3QEVYRK)
## Verification Evidence
{"command":"go test ./...","exit_code":0,"duration_ms":71263,"artifact_hash":"sha256:86273de464b330ee695c0eb3c2c73690fbc8f11b719474672bd91bdd3d65b189","verifier":"a-root","branch":"main","commit_sha":"ee01a9a5afdfe867a60d4dfbeb6c74b3881df0e8"}
