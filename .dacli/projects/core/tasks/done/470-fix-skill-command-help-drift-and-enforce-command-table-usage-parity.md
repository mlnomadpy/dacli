---
id: t-01M0AF65RDNBEX2SEF9JC5RTMZ
kind: task
created: 2026-08-18T12:57:50Z
created_by: a-root
owner: a-root
priority: should
github:
  issue: 692
  repo: mlnomadpy/dacli
estimate: "{optimistic: 3, probable: 5, pessimistic: 8}"
---
# Fix skill command help drift and enforce command-table usage parity
## Context
Adopted from GitHub issue #692.

## Reproduction

```text
$ dacli skill add --help
dacli skill add

Author a workspace skill

dacli queue add <slug> --step 'cmd or instruction'... [--title t]
```

The handler's real usage is `dacli skill add <name> --desc ... [--body ...] [--min-delivery ...]`. Source inspection also shows `skill promote` advertising the unrelated `shortcut promote` signature. The command table is the shared CLI/MCP/help contract, so these copy-pasted strings teach agents to call the wrong commands.



The same forward audit found two more malformed table entries in this task's
contract scope: `push` advertises the unrelated `github push` projection
command instead of the task-branch form accepted by `cmdPush`, and `accept`
contains a truncated quoted usage string. These must be corrected in this PR,
not left only as findings.

## Evidence

At audit time, the wrong literals are in `internal/features/skillforge/skillforge.go:21` and `:27`, while the correct handler contracts are at `:41` and `:88`.

## Acceptance
- [x] `dacli skill add --help` prints the exact flags accepted by `cmdAdd`.
- [x] `dacli skill promote --help` prints the exact lesson-ref/`--name` signature accepted by `cmdPromote`.
- [x] A table-driven invariant compares every command's declared `Usage` with the usage emitted by its handler on missing required arguments, allowing only explicitly documented variants.
- [x] The invariant covers CLI help and the MCP command description generated from the same command table.
- [x] Mutation evidence swaps one usage string and makes the invariant fail.
- [x] Focused CLI/skillforge/MCP tests and `go test ./...` pass.
- [x] `dacli push --help` prints the task-branch signature accepted by `cmdPush` and the invariant probes that handler.
- [x] `dacli accept --help` prints the full exact signature accepted by `cmdAccept` rather than a truncated quoted fragment.
- [x] The global command-path invariant fixes the additional reproduced `next`, `shortcut add`, and `template show` copy-paste drifts instead of leaving a finding that claims reverted edits landed.
- [x] `github pull` and `github sync` help advertise their implemented project, task-window, and `--dry-run` forms so the canonical operator preview flow agrees with current help.
## Log
- 2026-08-19T11:46:59Z claimed by a-fixer-yd1rff
- 2026-08-19T12:56:21Z accepted by a-root
- 2026-08-19T12:56:21Z verified by `GOCACHE=/tmp/dacli-gocache go test ./...` (exit 0) in branch main at 04afd39 — proves that tree builds, not that the work is in trunk
- 2026-08-19T12:56:21Z deliverable: dacli/470-fix-skill-command-help-drift-and-enforce-command-table-usage-parity is merged into main
- 2026-08-19T12:56:21Z completed by a-root
- 2026-08-19T13:30:33Z a-root: PR opened: https://github.com/mlnomadpy/dacli/pull/719 (event 01M0D07FKYG8BRYBCECFYGX245)
## Verification Evidence
{"command":"GOCACHE=/tmp/dacli-gocache go test ./internal/features/skillforge -run TestCmdAddUsageAndRejects","exit_code":0,"duration_ms":894,"artifact_hash":"sha256:8719ca943f9a008af887ac9462894dda6d885234599ddb1ae7f7b6da9e5636a9","verifier":"a-root","branch":"main","commit_sha":"04afd39d89aa6c49e06eefaeb7136bec6e09c4d8"}
{"command":"GOCACHE=/tmp/dacli-gocache go test ./internal/features/skillforge -run TestCmdPromoteUsageAndNotFound","exit_code":0,"duration_ms":423,"artifact_hash":"sha256:db51f5afde52c94f2beacab123e8d5fb7137041aaf25c9929e5a5b4f2362ef72","verifier":"a-root","branch":"main","commit_sha":"04afd39d89aa6c49e06eefaeb7136bec6e09c4d8"}
{"command":"GOCACHE=/tmp/dacli-gocache go test ./internal/cli -run TestCommandUsageMatchesHandlerUsage","exit_code":0,"duration_ms":1035,"artifact_hash":"sha256:e4cf1268a0d7897dbd258cd112a21560299cc35866ec0b8ac92062f1a9cb1662","verifier":"a-root","branch":"main","commit_sha":"04afd39d89aa6c49e06eefaeb7136bec6e09c4d8"}
{"command":"GOCACHE=/tmp/dacli-gocache go test ./...","exit_code":0,"duration_ms":80772,"artifact_hash":"sha256:2429a14634695bdd6836b37240aa6e5a404235e701037ac111cf850b45e201cd","verifier":"a-root","branch":"main","commit_sha":"04afd39d89aa6c49e06eefaeb7136bec6e09c4d8"}
{"command":"GOCACHE=/tmp/dacli-gocache go test ./internal/cli -run TestCommandUsageMatchesHandlerUsage","exit_code":0,"duration_ms":306,"artifact_hash":"sha256:8bb0155cf277c65f89f18ae1e57f20b850db2402d62e37241089f56e883ee098","verifier":"a-root","branch":"main","commit_sha":"04afd39d89aa6c49e06eefaeb7136bec6e09c4d8"}
{"command":"GOCACHE=/tmp/dacli-gocache go test ./internal/cli -run TestCommandUsageMatchesHandlerUsage","exit_code":0,"duration_ms":190,"artifact_hash":"sha256:8bb0155cf277c65f89f18ae1e57f20b850db2402d62e37241089f56e883ee098","verifier":"a-root","branch":"main","commit_sha":"04afd39d89aa6c49e06eefaeb7136bec6e09c4d8"}
{"command":"GOCACHE=/tmp/dacli-gocache go test ./internal/cli -run TestCommandUsageMatchesHandlerUsage","exit_code":0,"duration_ms":166,"artifact_hash":"sha256:8bb0155cf277c65f89f18ae1e57f20b850db2402d62e37241089f56e883ee098","verifier":"a-root","branch":"main","commit_sha":"04afd39d89aa6c49e06eefaeb7136bec6e09c4d8"}
{"command":"GOCACHE=/tmp/dacli-gocache go test ./internal/cli -run TestCommandUsageMatchesHandlerUsage","exit_code":0,"duration_ms":169,"artifact_hash":"sha256:8bb0155cf277c65f89f18ae1e57f20b850db2402d62e37241089f56e883ee098","verifier":"a-root","branch":"main","commit_sha":"04afd39d89aa6c49e06eefaeb7136bec6e09c4d8"}
{"command":"GOCACHE=/tmp/dacli-gocache go test ./...","exit_code":0,"duration_ms":2096,"artifact_hash":"sha256:07db5a7f494d86162ce19dd327d134bb8ac313296b8ab5376f50a81125baa384","verifier":"a-root","branch":"main","commit_sha":"04afd39d89aa6c49e06eefaeb7136bec6e09c4d8"}
