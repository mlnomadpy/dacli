---
id: t-01M0D3MKRKCHSX8P51HRDF0HQX
kind: task
created: 2026-08-19T13:33:44Z
created_by: a-root
owner: a-root
github:
  issue: 733
  repo: mlnomadpy/dacli
estimate: "{optimistic: 2, probable: 3, pessimistic: 5}"
---
# Make logs accept the documented separated --tail N form
## Context
Adopted from GitHub issue #733.

## Reproduction

Current help advertises:

```text
dacli logs <run-id-prefix|child-id> [-f] [--tail N]
```

But the documented invocation fails:

```text
$ dacli logs 01M0D2RR4M --tail 8
dacli: --tail must be an integer, got "true"
```

`dacli logs 01M0D2RR4M --tail=8` succeeds. This is especially costly for agents because the canonical skill teaches command signatures as executable contracts.

## Acceptance
- [x] `dacli logs <run> --tail 8` and `--tail=8` both select the same last eight transcript lines.
- [x] Missing, zero, negative, and non-integer tail values return the documented usage/exit-code behavior.
- [x] A public CLI regression covers the exact separated form printed by `--help`.
- [x] The shared flag parser is audited for other value-taking flags with the same help/behavior mismatch; evidence-backed siblings are fixed or filed separately.
- [x] Mutation evidence and `go test ./...` pass.
## Log
- 2026-08-27T13:07:48Z claimed by a-fixer-0hn8n8
- 2026-08-27T13:25:22Z a-root: PR opened: https://github.com/mlnomadpy/dacli/pull/829 (event 01M11NW61YJV23VBVRK7AWXXKH)
- 2026-08-27T13:28:46Z accepted by a-root
- 2026-08-27T13:28:46Z verified by `GOCACHE=/tmp/dacli-go-cache-487 go test ./...` (exit 0) in branch dacli/487-make-logs-accept-the-documented-separated-tail-n-form at 2aacbcf1 — proves that tree builds, not that the work is in trunk
- 2026-08-27T13:28:46Z deliverable: dacli/487-make-logs-accept-the-documented-separated-tail-n-form is merged into main
- 2026-08-27T13:28:46Z completed by a-root
## Verification Evidence
{"command":"GOCACHE=/tmp/dacli-go-cache-487 go test ./...","exit_code":0,"duration_ms":1554,"artifact_hash":"sha256:07db5a7f494d86162ce19dd327d134bb8ac313296b8ab5376f50a81125baa384","verifier":"a-root","branch":"dacli/487-make-logs-accept-the-documented-separated-tail-n-form","commit_sha":"4593482339534937d9e8d44d49f8124fe7625c99"}
{"command":"GOCACHE=/tmp/dacli-go-cache-487 go test ./...","exit_code":0,"duration_ms":1599,"artifact_hash":"sha256:07db5a7f494d86162ce19dd327d134bb8ac313296b8ab5376f50a81125baa384","verifier":"a-root","branch":"dacli/487-make-logs-accept-the-documented-separated-tail-n-form","commit_sha":"2aacbcf139911bd0825633c5a02e67932359def2"}
