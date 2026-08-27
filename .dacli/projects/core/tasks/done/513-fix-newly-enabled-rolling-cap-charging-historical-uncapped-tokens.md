---
id: t-01M11G1NEZZDAXBX18A8W98BQG
kind: task
created: 2026-08-27T11:35:23Z
created_by: a-root
owner: a-root
estimate: "{optimistic: 1, probable: 2, pessimistic: 3}"
github:
  issue: 812
  repo: mlnomadpy/dacli
---
# Fix newly enabled rolling cap charging historical uncapped tokens
## Acceptance
- [x] A restored governor with zero window_start and nonzero historical spend starts a fresh window at the first capped invocation and does not charge pre-window tokens.
- [x] A restored governor with an active nonzero window_start preserves its spend and still refuses when the configured ceiling is exhausted.
- [x] Dry-run and real loop decisions use the same initialization rule without dry-run persisting state.
- [x] Mutation evidence and the full repository verification gates pass.
## Log
- 2026-08-27T11:50:54Z accepted by a-root
- 2026-08-27T11:50:54Z verified by `GOCACHE=/private/tmp/dacli-go-cache-513 go test ./...` (exit 0) in branch main at 0d90b9a — proves that tree builds, not that the work is in trunk
- 2026-08-27T11:50:54Z deliverable: dacli/513-fix-newly-enabled-rolling-cap-charging-historical-uncapped-tokens is merged into main
- 2026-08-27T11:50:54Z completed by a-root
- 2026-08-27T11:54:53Z a-root: PR opened: https://github.com/mlnomadpy/dacli/pull/815 (event 01M11GF95J0CEGP1ZGY9FEFE3D)
## Verification Evidence
{"command":"GOCACHE=/private/tmp/dacli-go-cache-513 go test ./...","exit_code":0,"duration_ms":1730,"artifact_hash":"sha256:07db5a7f494d86162ce19dd327d134bb8ac313296b8ab5376f50a81125baa384","verifier":"a-root","branch":"dacli/513-fix-newly-enabled-rolling-cap-charging-historical-uncapped-tokens","commit_sha":"69d73e1f273110da9ddf501a7fbc92e038582d9d"}
{"command":"GOCACHE=/private/tmp/dacli-go-cache-513 go test ./...","exit_code":0,"duration_ms":1637,"artifact_hash":"sha256:07db5a7f494d86162ce19dd327d134bb8ac313296b8ab5376f50a81125baa384","verifier":"a-root","branch":"main","commit_sha":"0d90b9a676551352cdff129d7e178262692526c0"}
