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
- [ ] `dacli logs <run> --tail 8` and `--tail=8` both select the same last eight transcript lines.
- [ ] Missing, zero, negative, and non-integer tail values return the documented usage/exit-code behavior.
- [ ] A public CLI regression covers the exact separated form printed by `--help`.
- [ ] The shared flag parser is audited for other value-taking flags with the same help/behavior mismatch; evidence-backed siblings are fixed or filed separately.
- [ ] Mutation evidence and `go test ./...` pass.
## Log
