---
id: t-01M12K8SH454ZH3Z1MB1Q3D4TG
kind: task
created: 2026-08-27T21:50:57Z
created_by: a-root
owner: a-root
priority: must
estimate: "{optimistic: 2, probable: 3, pessimistic: 5}"
---
# Normalize or reject legacy blocks dependencies consistently
## Acceptance
- [x] task add handles :blocks consistently with task depend: either normalize it to FS before persistence or reject it with FS/SS/FF/SF guidance.
- [x] Existing stored :blocks aliases do not prevent unrelated dependency edits or critical-path analysis; behavior is covered by a migration or targeted compatibility regression.
- [x] A round-trip regression proves no newly created task persists an unsupported dependency type.
- [x] go test ./... passes.
## Log
- 2026-08-27T21:53:05Z claimed by a-fixer-dqsb6g
- 2026-08-27T22:11:05Z a-root: PR opened: https://github.com/mlnomadpy/dacli/pull/836 (event 01M12MC9AKGWCARXAP4ED1WKWJ)
- 2026-08-27T22:20:05Z accepted by a-root
- 2026-08-27T22:20:05Z verified by `env GOCACHE=/tmp/dacli-accept-go-cache go test ./...` (exit 0) in branch main at 483b32d4 — proves that tree builds, not that the work is in trunk
- 2026-08-27T22:20:05Z deliverable: dacli/526-normalize-or-reject-legacy-blocks-dependencies-consistently is merged into main
- 2026-08-27T22:20:05Z completed by a-root
## Verification Evidence
{"command":"env GOCACHE=/tmp/dacli-accept-go-cache go test ./...","exit_code":0,"duration_ms":2092,"artifact_hash":"sha256:07db5a7f494d86162ce19dd327d134bb8ac313296b8ab5376f50a81125baa384","verifier":"a-root","branch":"main","commit_sha":"483b32d4db733ff4dace381474773ff2bb2babb0"}
