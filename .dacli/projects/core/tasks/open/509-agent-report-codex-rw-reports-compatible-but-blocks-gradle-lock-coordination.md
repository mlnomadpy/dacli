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
- [ ] A deterministic adapter fixture reproduces a workspace-write runtime that starts successfully but cannot create the local coordination socket required by a declared Gradle verification command.
- [ ] Runtime/project preflight distinguishes generic CLI startup compatibility from build-tool capability and never labels the Gradle workflow verified from startup alone.
- [ ] A Gradle project either receives a documented compatible execution contract or fails closed before worker spend with the blocked socket capability and recovery route named.
- [ ] Codex, Claude, Gemini, Copilot, and generic runtimes continue to share provider-neutral capability semantics without scheduler-side vendor branching.
- [ ] Mutation evidence and the full repository verification gates pass on macOS and Linux fixtures without requiring a real Android SDK.
## Log
- 2026-08-27T22:42:52Z claimed by a-maintainer-ptwdk2
