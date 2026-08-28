---
id: t-01M146B9WDE7025DPT49EQKV0J
kind: task
created: 2026-08-28T12:43:36Z
created_by: a-root
owner: a-root
github:
  issue: 861
  repo: mlnomadpy/dacli
estimate: "{optimistic: 3, probable: 5, pessimistic: 8}"
depends_on: "[t-01M11HZWFC5EG8DPVSWWF8XV6B]"
---
# Expose a generated CLI capability manifest and skill compatibility check
## Context
Adopted from GitHub issue #861.

## Parent

Part of #855. Related concrete documentation drift: #807.

## Observed symptom

Skills and generated operating guidance can recommend commands or flags unsupported by the installed dacli binary. Runtime capability reporting exists, but there is no stable machine-readable manifest for the CLI/MCP surface and compatibility schemas.

## Objective

Expose a generated capability manifest and compatibility diagnosis:

```bash
dacli capabilities --json
dacli version --compatibility
```

## Required manifest

- CLI build/version and state/schema versions.
- Commands, subcommands, aliases, flags, JSON support, mutation classification, and usage contract from the central registry.
- MCP schema/tool versions.
- Prompt/override schema versions.
- Runtime adapter capability versions, separated from CLI capabilities.
- Stable capability identifiers suitable for `requires`/`optional` declarations in skills and generated guidance.



## Non-goals

- Automatically downloading or replacing binaries.
- Treating runtime/model availability as proof of authentication.
- Maintaining a second handwritten compatibility matrix.

## Manual workaround today

Agents probe individual `--help` commands and compare skill text with the installed binary at runtime.

## Acceptance
- [ ] The JSON manifest is generated from authoritative registries rather than a hand-maintained duplicate command list.
- [ ] Golden tests prove every registered command/flag and MCP tool appears with stable identifiers and version/schema metadata.
- [ ] `version --compatibility` accepts or discovers a skill capability requirement document and reports supported, optional-missing, required-missing, and incompatible-schema states.
- [ ] Required missing capabilities fail closed with the CLI exit-code contract; optional missing capabilities include an actionable fallback.
- [ ] A regression fixture for #807 prevents guidance from emitting `task check --verify` when the installed manifest lacks it.
- [ ] Binary identity/path and the version that generated installed skills/state are reported without leaking unrelated filesystem data.
- [ ] Tests fail when a registry command is omitted from the manifest or a removed flag remains advertised.
## Log
- 2026-08-28T12:44:47Z dependency edit by a-root (event 01M146DF5Y68F43MTF8E7WNH89)
