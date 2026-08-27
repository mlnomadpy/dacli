---
id: t-01M1068MEG379NZ2SE5EH6DYZC
kind: task
created: 2026-08-26T23:25:11Z
created_by: a-root
owner: a-root
github:
  issue: 799
  repo: mlnomadpy/dacli
estimate: "{optimistic: 5, probable: 8, pessimistic: 13}"
---
# [agent-report] codex-rw reports compatible but blocks Gradle lock coordination sockets
## Context
Adopted from GitHub issue #799.

On macOS arm64, runtime doctor --grant rw reports the Codex workspace-write adapter as compatible. A spawned Android worker then fails before Gradle configuration because Gradle cannot create its file-lock coordination socket (java.net.SocketException: Operation not permitted). The same Gradle test and assemble command succeeds immediately outside the worker sandbox. Adding both the repository Git metadata directory and the Gradle cache with --add-dir does not help. The codex-rw preset should either provide a build-capable writable contract for this standard workflow or detect and report that the runtime cannot verify Gradle projects.

---
_Reported via `dacli report`._
- dacli: dev
- platform: darwin/arm64
- workspace and run transcript withheld (public upstream) — re-run with --disclose to include them

## Acceptance
- [x] A deterministic adapter fixture reproduces a workspace-write runtime that starts successfully but cannot create the local coordination socket required by a declared Gradle verification command.
- [x] Runtime/project preflight distinguishes generic CLI startup compatibility from build-tool capability and never labels the Gradle workflow verified from startup alone.
- [x] A Gradle project either receives a documented compatible execution contract or fails closed before worker spend with the blocked socket capability and recovery route named.
- [x] Codex, Claude, Gemini, Copilot, and generic runtimes continue to share provider-neutral capability semantics without scheduler-side vendor branching.
- [x] Mutation evidence and the full repository verification gates pass on macOS and Linux fixtures without requiring a real Android SDK.
## Log
- 2026-08-27T22:42:52Z claimed by a-maintainer-ptwdk2
- 2026-08-27T23:02:35Z a-root: PR opened: https://github.com/mlnomadpy/dacli/pull/840 (event 01M12Q7WSSH8P8BS31SBMKJWRG)
- 2026-08-27T23:08:55Z accepted by a-root
- 2026-08-27T23:08:55Z verified by `env GOCACHE=/tmp/dacli-accept-509-go-cache go test ./...` (exit 0) in branch dacli/509-agent-report-codex-rw-reports-compatible-but-blocks-gradle-lock-coordination at bb892018 — proves that tree builds, not that the work is in trunk
- 2026-08-27T23:08:55Z deliverable: dacli/509-agent-report-codex-rw-reports-compatible-but-blocks-gradle-lock-coordination exists but is NOT in main — closed anyway
- 2026-08-27T23:08:55Z completed by a-root
- 2026-08-27T23:12:37Z a-root: Landing policy override: mode=pr base=main (event 01M12QQS02073T43W5P3FX3NMS)
- 2026-08-27T23:12:37Z a-root: Integrated via PR https://github.com/mlnomadpy/dacli/pull/840 at merge commit 40d6f27ea58dddc939ce874bf9e6518f634b0c0b into main (generation 0) (event 01M12QQZW4TXN3YJEXVWDXQDZD)
## Verification Evidence
{"command":"env GOCACHE=/tmp/dacli-509-check-go-cache go test ./internal/store ./internal/features/execution ./internal/features/orchestration","exit_code":0,"duration_ms":40147,"artifact_hash":"sha256:df8d0bb0a6548c7235c605a54a0dcb40c1e464b0283cdb24bf4510ab079ce32d","verifier":"a-root","branch":"dacli/509-agent-report-codex-rw-reports-compatible-but-blocks-gradle-lock-coordination","commit_sha":"bb892018a7192df888373af8da3755bfe6241f52"}
{"command":"env GOCACHE=/tmp/dacli-accept-509-go-cache go test ./...","exit_code":0,"duration_ms":2261,"artifact_hash":"sha256:07db5a7f494d86162ce19dd327d134bb8ac313296b8ab5376f50a81125baa384","verifier":"a-root","branch":"dacli/509-agent-report-codex-rw-reports-compatible-but-blocks-gradle-lock-coordination","commit_sha":"bb892018a7192df888373af8da3755bfe6241f52"}
