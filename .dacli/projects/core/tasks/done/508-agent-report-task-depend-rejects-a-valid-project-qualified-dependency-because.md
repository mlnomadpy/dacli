---
id: t-01M1068M8HJ9G8XCXMEMVE2V8D
kind: task
created: 2026-08-26T23:25:11Z
created_by: a-root
owner: a-root
github:
  issue: 800
  repo: mlnomadpy/dacli
estimate: "{optimistic: 3, probable: 5, pessimistic: 8}"
---
# [agent-report] task depend rejects a valid project-qualified dependency because unrelated legacy tasks contain ambiguous unqualified refs. Repro: adding an FS dependency between two project-qualified tasks aborts while validating unrelated stored dependencies such as 002 or 014 across other projects. Expected: validate the changed task/edge or resolve stored refs within their owning project; unrelated legacy ambiguity must not prevent all future graph edits.
## Context
Adopted from GitHub issue #800.

---
_Reported via `dacli report`._
- dacli: dev
- platform: darwin/arm64
- workspace and run transcript withheld (public upstream) — re-run with --disclose to include them

## Acceptance
- [x] A multi-project fixture contains ambiguous legacy numeric dependency refs unrelated to the task being changed.
- [x] `task depend <project-qualified-task> --on <project-qualified-task>` validates and persists the requested edge without resolving unrelated legacy refs globally.
- [x] Cycle detection and missing-target validation still reject invalid edges within the changed task's reachable dependency graph.
- [x] Existing legacy refs resolve in their owning project where possible and produce a scoped diagnostic when that specific edge is inspected.
- [x] Mutation evidence and the full repository verification gates pass.
## Log
- 2026-08-28T00:15:57Z claimed by a-maintainer-z0nyk9
- 2026-08-28T00:28:34Z a-root: PR opened: https://github.com/mlnomadpy/dacli/pull/844 (event 01M12VXGCA4N10XS8CBXHRPE1M)
- 2026-08-28T00:31:13Z accepted by a-root (applied 1 proposal(s))
- 2026-08-28T00:31:13Z verified by `env GOCACHE=/tmp/dacli-root-508-accept-cache go test ./...` (exit 0) in branch dacli/508-agent-report-task-depend-rejects-a-valid-project-qualified-dependency-because at 836579ef — proves that tree builds, not that the work is in trunk
- 2026-08-28T00:31:13Z deliverable: dacli/508-agent-report-task-depend-rejects-a-valid-project-qualified-dependency-because exists but is NOT in main — closed anyway
- 2026-08-28T00:31:13Z completed by a-root
- 2026-08-28T00:44:43Z a-root: Landing policy override: mode=pr base=main (event 01M12WEW2XN8KKV6NNCY4ETZ55)
- 2026-08-28T00:44:43Z a-root: Integrated via PR https://github.com/mlnomadpy/dacli/pull/844 at merge commit c3285d40a954be950b53097a6f07b2ba6bac0b96 into main (generation 0) (event 01M12WF3C3T1H96E4Q5VFXPG4X)
## Verification Evidence
{"command":"env GOCACHE=/tmp/dacli-root-508-cache go test ./internal/store -run TestDependencyChangeIgnoresUnrelatedAmbiguousLegacyRefs","exit_code":0,"duration_ms":649,"artifact_hash":"sha256:71fe97afcc65b95ce716b2ff4eeee357626295a509a9f03ec81f6d12f2a1fb02","verifier":"a-root","branch":"dacli/508-agent-report-task-depend-rejects-a-valid-project-qualified-dependency-because","commit_sha":"836579efd903fc3a6601048a194679ad9eeeee38"}
{"command":"env GOCACHE=/tmp/dacli-root-508-cache go test ./internal/store -run TestDependencyChange","exit_code":0,"duration_ms":1034,"artifact_hash":"sha256:f9cd02928c7aa2784a458545b5a1775838b92cdafd94d349aeae4b22a86fc79b","verifier":"a-root","branch":"dacli/508-agent-report-task-depend-rejects-a-valid-project-qualified-dependency-because","commit_sha":"836579efd903fc3a6601048a194679ad9eeeee38"}
{"command":"env GOCACHE=/tmp/dacli-root-508-cache go test ./internal/store -run TestReadyFrontierScopesAmbiguousLegacyRefToOwningTask","exit_code":0,"duration_ms":515,"artifact_hash":"sha256:c7c146ee473765499656f308695a38f1517c3a48ebea72c55f3212a072c5138c","verifier":"a-root","branch":"dacli/508-agent-report-task-depend-rejects-a-valid-project-qualified-dependency-because","commit_sha":"836579efd903fc3a6601048a194679ad9eeeee38"}
{"command":"env GOCACHE=/tmp/dacli-root-508-accept-cache go test ./...","exit_code":0,"duration_ms":94127,"artifact_hash":"sha256:ce741cb1539c3d8fd59b8e560ed8e5354beb79bdfee17e6e0316a970bcdf9a3d","verifier":"a-root","branch":"dacli/508-agent-report-task-depend-rejects-a-valid-project-qualified-dependency-because","commit_sha":"836579efd903fc3a6601048a194679ad9eeeee38"}
