---
id: t-01M0AEYFXAB22RE9Y2SH9WZZKR
kind: task
created: 2026-08-18T12:53:38Z
created_by: a-root
owner: a-root
priority: should
github:
  issue: 691
  repo: mlnomadpy/dacli
estimate: "{optimistic: 8, probable: 13, pessimistic: 21}"
---
# Make runtime context provenance and configuration isolation explicit across coding CLIs
## Context
Adopted from GitHub issue #691.

## Reproduction

Two independent governed Codex runs on 2026-08-18 began with:

```text
ERROR ... failed to load skill /Users/.../.agents/skills/math-ml-paper-writing/SKILL.md:
missing YAML frontmatter delimited by ---
```

The configured adapter already passes `--ignore-user-config`, disables plugins/plugin sharing/remote plugins, and uses `--ephemeral`, yet the child still discovers an operator-global skill outside the repository and outside the role's declared `skills:` list. The run continued, so `runtime doctor` and `preflight` did not surface this undeclared context source.

This is not only a Codex flag problem. Claude Code, Gemini CLI, Copilot CLI, and generic executors each have different configuration, instruction, extension, MCP, and skill discovery rules. The runtime contract currently records prompt/model/result/usage/sandbox/exit behavior but not configuration provenance or hermeticity.

## Risk

An undeclared global skill/plugin/config can alter behavior, add instructions, consume context, fail loading, or introduce tools/network access that the role and run record do not name. Reproducing a run on another machine becomes impossible, and a claimed provider-independent verification panel may actually share the same operator-global instructions.

## Design

Add a provider-neutral context-provenance contract to runtime adapters. Each preset declares which external configuration classes it can disable, isolate, enumerate, or cannot control. `runtime doctor` behaviorally probes the supported isolation mode; `preflight` compares the effective sources with the role's declared skills/tools. Unattended strict mode fails closed on undeclared sources, while an explicit cooperative/allow-user-config mode records the exception and the discovered source list in `invocation.txt`.



## Manual workaround

Repair or temporarily move the operator-global skill and inspect every transcript header. This does not prove that no other global instruction/plugin source was loaded.

## Acceptance
- [ ] Runtime schema represents user config, repository instructions, global skills, plugins/extensions, MCP servers, and environment-derived config as isolated, enumerated, allowed, or unsupported capabilities.
- [ ] Codex, Claude Code, Gemini CLI, Copilot CLI, and generic presets document and test their effective discovery/isolation behavior without assuming a common flag.
- [ ] `runtime doctor` behaviorally detects at least one deliberately invalid global fixture for each runtime that claims isolation; accepting a flag in help text alone is insufficient.
- [ ] `preflight` reports every undeclared context/tool source and strict unattended spawn refuses when the runtime cannot isolate or enumerate it.
- [ ] An explicit cooperative/allow-user-config override records the exact source paths/classes in `invocation.txt` and emits a visible warning.
- [ ] The run record includes the declared role skills and the effective external-context provenance, never secret values.
- [ ] Verification panels can require independent context provenance in addition to different runtime/model names.
- [ ] Tests use fixture homes/config roots and do not depend on the developer's real global configuration.
- [ ] Mutation evidence, focused runtime/execution tests, and `go test ./...` pass.
## Log
