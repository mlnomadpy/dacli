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
estimate: "{optimistic: 1, probable: 2, pessimistic: 3}"
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



## Evidence

At audit time, the wrong literals are in `internal/features/skillforge/skillforge.go:21` and `:27`, while the correct handler contracts are at `:41` and `:88`.

## Acceptance
- [ ] `dacli skill add --help` prints the exact flags accepted by `cmdAdd`.
- [ ] `dacli skill promote --help` prints the exact lesson-ref/`--name` signature accepted by `cmdPromote`.
- [ ] A table-driven invariant compares every command's declared `Usage` with the usage emitted by its handler on missing required arguments, allowing only explicitly documented variants.
- [ ] The invariant covers CLI help and the MCP command description generated from the same command table.
- [ ] Mutation evidence swaps one usage string and makes the invariant fail.
- [ ] Focused CLI/skillforge/MCP tests and `go test ./...` pass.
## Log
