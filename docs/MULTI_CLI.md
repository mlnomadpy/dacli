# Operating multiple CLIs and routing models

This runbook is for an operator who has more than one coding-agent CLI or more
than one model tier. It covers the safe path from adapter setup to a spawn, and
the controls that keep expensive models and provider limits available for the
work that needs them. For adapter internals and the executable conformance
matrix, see [RUNTIMES.md](RUNTIMES.md).

## 1. Choose the read-only or writer preset

dacli ships nine presets. Four providers have a read-only/writer pair; the
unpaired `generic-exec` preset is the base for an operator-defined adapter.

| Provider | Read-only preset | Read-write preset | Boundary |
|---|---|---|---|
| Claude Code | `claude-code` | `claude-code-rw` | The reader allowlists Read/Grep/Glob/LS and dacli reporting. The writer adds Edit/Write plus scoped git and dacli shell commands. |
| Codex | `codex` | `codex-rw` | The reader uses `--sandbox read-only`; the writer uses `workspace-write` and retains the read-only override for `ro` spawns. |
| Gemini CLI | `gemini` | `gemini-rw` | The reader uses approval mode `plan`; the writer uses `auto_edit`, not unrestricted yolo mode. |
| GitHub Copilot CLI | `copilot` | `copilot-rw` | The reader denies write and shell; the writer grants write and only scoped git/dacli shell commands. |
| Any executable | — | `generic-exec` | No binary, sandbox, or structured-output contract is assumed. Configure and verify the executable yourself; it is not a read-only safety claim. |

The plain provider preset is intended for reviewers and auditors. Use the
`-rw` counterpart for implementers. A writer preset can still serve an `ro`
spawn because dacli applies its read-only sandbox arguments for that grant.

## 2. Set up and verify adapters

Authenticate each vendor CLI using its own login flow first. dacli records the
binary, flags, and a narrow environment allowlist; it does not store provider
credentials.

Codex is a complete example:

```bash
dacli runtime add codex --preset codex
dacli runtime add codex-rw --preset codex-rw
dacli runtime list
dacli runtime doctor
```

`runtime doctor` uses local version/help and sandbox probes wherever those are
sufficient. Codex exec adapters additionally send one trivial, versioned JSONL
startup request and stop at `turn.started`; they do not wait for the model
answer. A compatible result is cached against the exact binary, adapter,
grant, model, and strategy version. Do not treat a declared sandbox as
verified: an `ro` spawn requires a current verified probe for the installed
binary and exact flags.

The equivalent setup for the other shipped adapters is concise:

```bash
dacli runtime add claude --preset claude-code
dacli runtime add claude-rw --preset claude-code-rw
dacli runtime add gemini --preset gemini
dacli runtime add gemini-rw --preset gemini-rw
dacli runtime add copilot --preset copilot
dacli runtime add copilot-rw --preset copilot-rw
dacli runtime add local-agent --preset generic-exec --binary my-agent
dacli runtime doctor
```

For `generic-exec`, add the executable's real prompt, model, sandbox, and output
flags with `runtime add` overrides. Start from the vendor's documented
non-interactive invocation, then inspect the result with `runtime list` and
`runtime doctor`; dacli deliberately makes no vendor safety claim for it.

## 3. Define provider-neutral roles and models

A role binds work policy to a runtime and model. The model profile describes
selection facts without vendor names:

```bash
dacli role add codex-reviewer --kind reviewer --grant ro \
  --runtime codex --model-id gpt-5.4-mini --cost-tier 2 \
  --max-task-points 3 --context-limit 200000 --capability-tag code

dacli role add codex-fixer --kind implementer --grant rw \
  --runtime codex-rw --model-id gpt-5.4 --cost-tier 5 \
  --max-task-points 8 --context-limit 200000 --capability-tag code
```

`model_id` is passed through the runtime's model flag. `cost_tier` is an
ordering rank, not a price; lower is cheaper. `max_task_points` is the largest
task estimate the role may take, `context_limit` records its declared context
capacity, and `capability_tags` state requirements such as code or vision.

After estimating a task, ask dacli for the cheapest eligible role whose
capacity covers expected points (`Te`):

```bash
dacli task estimate 415 --estimate 2,3,5
dacli team assign 415
```

The output names the inferred role kind, runtime, model, cost tier, capacity,
context, capabilities, and the proposed spawn command. If every cap is too
small, dacli refuses and tells you to decompose the task or add a heavier role.

Treat the recommendation as a cost-and-capacity answer, not a risk judgment.
Override it when blast radius, unfamiliar code, privacy, or required reasoning
is not represented by the profile:

```bash
# Choose a different vetted role (and its runtime/model policy).
dacli spawn --task 415 --role senior-fixer --advise

# One-run model override while retaining the role and runtime.
dacli spawn --task 415 --role codex-fixer --model gpt-5.4 --advise

# Explicit runtime override: dacli will not substitute a fallback for it.
dacli spawn --task 415 --role codex-fixer --runtime codex-rw --advise
```

`--advise` prints calibrated sizing and taint status and exits without launching
or billing a run. Re-run without it only after the preview is acceptable.

## 4. Preflight, then spawn

For Codex, verify the exact role/runtime/grant combination before launch:

```bash
dacli preflight --role codex-reviewer --runtime codex --grant ro
dacli spawn --task 415 --role codex-reviewer --grant ro --worktree --detach

dacli preflight --role codex-fixer --runtime codex-rw --grant rw
dacli spawn --task 415 --role codex-fixer --grant rw \
  --worktree --claim docs --detach --max-tokens 40000
```

Equivalent provider commands differ only by the configured role (or runtime):

```bash
dacli preflight --role claude-fixer && dacli spawn --task 415 --role claude-fixer --worktree --detach
dacli preflight --role gemini-reviewer && dacli spawn --task 415 --role gemini-reviewer --worktree --detach
dacli preflight --role copilot-fixer && dacli spawn --task 415 --role copilot-fixer --worktree --detach
dacli preflight --role local-agent && dacli spawn --task 415 --role local-agent --worktree --detach
```

Preflight reports every mismatch it can observe: grant versus write capability,
the running dacli binary versus the runtime allowlist, and tools named by the
role prompt versus the adapter. `--cooperative` is an explicit weakening for an
adapter whose sandbox dacli cannot enforce; use it only when that trust decision
is intentional.

## 5. Provider limits, breakers, and explicit fallback

dacli classifies provider failures into `rate_limit`, `quota_exhausted`,
`authentication`, `unavailable`, `permanent_input`, `policy_refusal`, and
`unknown`. Rate limits and temporary unavailability are retryable. Rate limits,
quota exhaustion, and unavailability may open a fallback. Authentication,
invalid input, an unknown failure, and a policy refusal do not silently move
work elsewhere.

For a rate limit or temporary outage, the cooldown uses the provider's reset
interval when reported, otherwise a bounded retry-policy delay. A fallbackable
failure opens a persisted circuit breaker under
`.dacli/runs/runtime-cooldowns/`, so restarting the loop does not hammer the
same provider again. Operator output and `transitions.log` record the same line:

```text
source=codex-rw destination=claude-rw reason=rate limit cooldown=1h0m0s
```

Fallbacks are opt-in and ordered. Add role names to the source role file's
frontmatter; dacli persists the field as `fallback_to`:

```yaml
fallback_to:
  - claude-fixer
  - gemini-fixer
```

Only this named chain is considered. Each destination must preserve the source
grant and every required capability tag, and its runtime circuit must be
closed. dacli never guesses a provider. An explicit `--runtime` is also never
substituted: remove that override and route through the role if fallback is
desired. If no eligible destination remains, spawn returns exit 3 with the
source runtime's remaining cooldown.

## 6. Conserve limits and cost

- Run `runtime doctor` and `preflight` before spawning; both are observable,
  credential-free checks that prevent paid runs from discovering flag drift.
- Use `team assign` and `--advise`, then keep routine work on the lowest cost
  tier whose capacity and capabilities fit. Override upward for consequence,
  not habit.
- Give each launch `--max-tokens`; use loop-level `--window-tokens` with
  `--budget-window` for a rolling provider budget. `dacli calibrate` improves
  estimates from recorded usage.
- Prefer read-only roles for review and narrow claims for writers. Decompose
  work beyond a role's capacity instead of spending a large-context model on a
  task that is too broad.
- Do not retry a refusal or bypass an open circuit with repeated explicit
  runtime spawns. Wait for its printed cooldown or deliberately select another
  verified role.

## 7. Troubleshoot from observable state

```bash
dacli runtime list
dacli runtime doctor
dacli preflight --role codex-fixer
dacli team assign 415
dacli agents --tail
dacli logs <run-id> --tail 80
dacli runs show <run-id>
dacli doctor
```

Interpret command exits consistently:

| Exit | Meaning | Operator action |
|---|---|---|
| 0 | The command completed. | Continue; for `--advise`, remember that no child was launched. |
| 1 | Runtime or operational failure. | Read the diagnostic and transcript; repair the install, authentication, or provider condition. |
| 2 | Usage error. | Correct the command or unknown flag. |
| 3 | Policy refusal. | Stop retrying. Follow the named remedy: verify the sandbox, choose an eligible role, decompose, or wait for the circuit. |
| 4 | Object not found. | Check the task, role, runtime, or run reference. |

Common cases are intentionally visible: “binary not on PATH” points to
`runtime doctor`; an unknown or failed read-only probe refuses an `ro` spawn;
an rw role on a reader preset fails preflight; a model flag missing from the
installed CLI appears as doctor flag drift; and a provider transition is both
printed and retained beside the cooldown records. Fix the stated condition
rather than changing grants or weakening assertions merely to obtain exit 0.
