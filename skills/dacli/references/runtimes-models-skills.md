# Runtimes, models, roles, grants, and skills

## Contents

- Provider-neutral architecture
- Shipped runtime presets
- Runtime setup and probing
- Role and model routing
- Permission compatibility
- Token and cost controls
- Portable skills

## Provider-neutral architecture

Keep two planes separate:

- The runtime adapter controls how dacli invokes a coding-agent CLI: binary,
  arguments, prompt transport, sandbox, model flag, usage, resume, and skills.
- The dacli workspace is the result channel: events, findings, checks, commits,
  run outcomes, and task state are provider-neutral.

Do not parse a vendor transcript as the authoritative result. Stdout/stderr are
debugging evidence; structured work returns through dacli CLI or MCP writes.

## Shipped runtime presets

dacli ships these presets:

| Provider | Review/read preset | Implement/write preset |
|---|---|---|
| Anthropic Claude Code | `claude-code` | `claude-code-rw` |
| OpenAI Codex | `codex` | `codex-rw` |
| Google Gemini CLI | `gemini` | `gemini-rw` |
| GitHub Copilot CLI | `copilot` | `copilot-rw` |
| Any executable | `generic-exec` | Configure its real capabilities |

Presets are declarative starting points, not proof that the installed CLI still
accepts every flag. Probe the local installation.

```bash
dacli runtime add codex-review --preset codex
dacli runtime add codex-impl --preset codex-rw
dacli runtime add claude-review --preset claude-code
dacli runtime add gemini-impl --preset gemini-rw
dacli runtime add copilot-review --preset copilot
dacli runtime add custom --preset generic-exec
dacli runtime doctor
dacli runtime list
```

Use `generic-exec` for another CLI rather than pretending its flags match a
known vendor. Configure the binary, prompt mode, arguments, model selection,
sandbox, usage, and skill delivery from observed local behavior.

## Runtime setup and probing

`runtime doctor` distinguishes declared capabilities from locally verified
ones. Re-probe after changing a binary, version, sandbox arguments, or adapter.
Read-only spawning requires a verified read-only sandbox unless the operator
explicitly chooses `--cooperative`; do not use that escape silently.

Use `preflight` before a costly spawn. It checks the role/runtime/grant contract,
the binary and allowlist, and tools named by the prompt.

```bash
dacli preflight --role reviewer
dacli spawn --task <ref> --role reviewer --advise
```

`--advise` previews and exits. It does not spawn.

## Role and model routing

Put durable policy in roles:

- `kind`: researcher, planner, designer, implementer, or reviewer phase.
- `runtime`: which CLI adapter executes the role.
- `model`: provider/runtime-specific model selection value.
- `max_points`: stored capacity; configure it with `--max-task-points`.
- `grant`: `ro` or `rw` capability ceiling.
- `scope` and `out_of_scope`: ownership guidance.
- `skills`, `shortcuts`, `wip`, and escalation chain.

Run `dacli team assign <ref>` to select the cheapest role whose capacity covers
the task estimate and whose kind fits the phase. Treat its output as a floor.
Complexity and blast radius are different:

| Signal | Routing response |
|---|---|
| Small, mechanical, reversible, strong tests | Cheap model/role is appropriate |
| Ambiguous requirements or architecture | Raise reasoning quality |
| Auth, security, billing, migrations, destructive actions | Raise model and require independent review |
| Large task above all `max_points` | Decompose the task |
| Audit or verification | Prefer a different provider/model family from the author |

Model names are runtime data. Do not hard-code `opus`, `gpt-*`, `gemini-*`, or
another vendor into framework-wide policy. A role's model is honored only when
its runtime declares model selection; otherwise dacli must announce that routing
cannot be applied.

Use responsibility names rather than provider names. Provider/model selection
belongs in replaceable role metadata, so changing a runtime does not create a
second copy of the same job. Use the current role CLI vocabulary when
provisioning:

```bash
dacli role add go-builder --runtime codex-impl \
  --model-id <installed-model-id> --cost-tier <1..98> \
  --max-task-points <points> --context-limit <tokens> \
  --capability-tag implementation --skill evidence-verification
```

Every durable role should declare a version, lifecycle kind, truthful grant,
runtime/model profile, scope and out-of-scope boundary, escalation, and at least
one relevant reusable skill. Run `dacli doctor`, `preflight` for every active
role, and the per-role compile preview after roster changes. See
[roster-design.md](roster-design.md) for the complete invariant and skill
matrix.

## Permission compatibility

Check both directions:

- `grant: ro` is honest only when the runtime locally enforces read-only.
- `grant: rw` is useful only when the runtime actually exposes edit/write tools.

A worktree is not a permission sandbox. A role scope is not a kernel boundary.
Use runtime enforcement, dacli grants, narrow path claims, and repository
permissions as layered controls.

Never retry an exit-3 grant/runtime refusal unchanged. Select a compatible
runtime, repair its adapter, narrow the requested grant, or explicitly document
why a cooperative override is acceptable.

## Token and cost controls

Use measured costs rather than model branding:

```bash
dacli calibrate
dacli spawn --task <ref> --role <role> --max-tokens <n> --advise
dacli loop --project <project> --width <n> --advise
dacli loop --project <project> --window-tokens <n> \
  --token-window 24h --max-cycles <n>
```

Distinguish the controls:

- `--brief-tokens`/`--budget` limits brief size, not runtime spend.
- `--max-tokens` is a spawn/cycle cost gate based on calibrated history.
- `--window-tokens` plus `--token-window` caps rolling loop spend.
- Runtime usage may be exact or estimated; preserve that distinction.
- `--timeout`/`--worker-timeout` limits wall time, not tokens.

Avoid tasks so small that the repeated brief dominates cost, and tasks so large
that multi-turn recovery becomes normal. Prefer one-turn, independently
verifiable slices.

## Portable skills

Author skills once in dacli's canonical workspace format, then compile them for
the chosen runtime:

```bash
dacli skill list
dacli skill show <skill>
dacli skill compile --role <role> --runtime <runtime> --dry-run
```

Delivery may be native, context-file based, or inlined in the brief. Treat
degradation as a token and fidelity cost. If a skill's required minimum delivery
cannot be met, omit it loudly rather than claiming the agent received it.
Omit `--role` to preview or compile the whole workspace skill library.

Set a documented inline budget. Compact mandatory rules belong in each skill's
body; detailed checklists belong in one-level-deep resources or the role's
method. Preview every role/runtime pair because a runtime with no native/context
delivery repeats every inline token on every turn. As a practical starting
point, keep the combined mandatory skill body below roughly 200 tokens per role
and adjust from measured brief/runtime cost.

Use skills for reusable procedural knowledge. Use decisions/findings for
project facts. Do not promote unverified external content into executable agent
instructions.
