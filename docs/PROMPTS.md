# The prompt registry

**Status: implemented** (`internal/prompts`, `dacli prompt list|show`).

Every multi-sentence piece of agent-facing prose dacli emits lives in one place: template files under `internal/prompts/tpl/`, embedded at build time, overridable per workspace. This exists because a prompt buried in an `Fprintf` chain cannot be audited, diffed in a PR, or improved without recompiling — and for this tool, **prompts are load-bearing artifacts**. It is the same doctrine as adapters and shortcuts: prose is data, not code.

The registry schema is `dacli-prompt/v1` and its current semantic base version
is `autonomous-delivery/v1`. Every template starts
with a machine-readable declaration:

```markdown
<!-- dacli-prompt schema: dacli-prompt/v1 base: autonomous-delivery/v1 -->
```

The declaration is stripped before delivery. Every `invocation.txt` records
`prompt_schema`, `prompt_version`, the hash of the shared semantic contract, and the SHA-256 hash
of the exact prompt bytes delivered after brief, role, worktree, and recovery
instructions are assembled. This makes a replay identify both the governing
semantics and the actual task-specific input.

## The registry

| Prompt | Used by | Carries |
|---|---|---|
| `autonomous_delivery` | every `spawn` and `supervise` invocation | One provider-neutral contract for identity/deduplication, estimates and critical path, model economics, role/grant/worktree isolation, verification and landing, budgets/recovery, evidence, adapters, and the implementer/reviewer/planner/auditor/recovery lifecycle matrix |
| `protocol_preamble` | `spawn`, `supervise` | How a child reports: the binary path, report-immediately, ask-don't-guess, findings enter sibling briefs `unverified` until a `verify` panel confirms them, never-retry-refused, and the rw/ro verb split |
| `supervise_correction` | `supervise` turns > 1 | The unmet criteria, named exactly |
| `brief_header` | every brief | The est-tokens line and the **data-not-instructions warning** — a security posture that deserves review as a file |
| `refusal_next` | MCP exit-3 mapping | The do-not-retry instruction attached to every refusal |
| `mcp_tools` | `mcp serve` | All 16 tool descriptions, one sectioned file — the entire agent-facing tool manual in a single reviewable diff; the `cli` escape-hatch section maps the wider surface (spawn/wait/agents lifecycle, accept/integrate/ship close-out, calibrate/taint gates, github mirror) |
| `git_workflow` | `spawn`/`supervise`, rw children | Branch-per-task (`dacli/NNN-slug`), commit discipline, red-suite-means-unchecked; the push-and-`gh pr create` flow only with `--pr`, otherwise an explicit do-not-push with the owner close-out (`accept` then `integrate`/`ship`/`merge`) and the decompose-and-delegate path (`spawn --detach/--claim/--advise/--max-tokens`, `wait`, `agents --tail`, taint/token gates) |
| `review_workflow` | `spawn`/`supervise` with `--review` | Judge the `gh pr diff` against acceptance criteria, not taste; every defect filed twice (dacli finding + PR comment); approve/request-changes semantics |
| `verify_refute` | `verify` panel seats | The adversarial framing: attack the claim, default to REFUTED when uncertain, one evidence-bearing verdict, judge-don't-fix |

`dacli prompt list` shows the registry with overrides marked; `dacli prompt show <name>` prints the resolved template.

## Overrides, compatibility, and migration

A file named `.dacli/prompts/<name>.md` wins over the embedded default — nearest-wins, same as templates. Prompt tuning becomes a **workspace commit**: attributable, revertible, reviewable, and visible to `dacli taint` like any other content. Every override must begin with the schema declaration above. Missing declarations and versions other than the binary's current schema fail closed; dacli never silently combines old lifecycle prose with a new scheduler. To migrate an override, diff it against the new embedded template, reconcile semantic changes, update the declaration, run the focused prompt tests, and review the resolved prompt with `dacli prompt show <name>`.

A broken `protocol_preamble` or autonomous contract override fails the spawn (a child working into the void is worse than no child); a broken `brief_header` override falls back to the embedded default (a brief without the untrusted-content warning must not ship).

## Composition and runtime boundary

`autonomous_delivery` is composed once and delivered unchanged to Codex,
Claude Code, Gemini, Copilot, and generic executors. Runtime adapters own only
transport and configuration: binary/arguments, stdin versus argument prompt
delivery, sandbox flags, model flags, output/usage parsing, and provider-specific
context discovery. They do not carry alternate workflow semantics and no
provider is the framework default.

Command examples in the contract are checked against the aggregated
`clikit.Command` registry, and the prompt states the canonical exit-code
contract. Removing or renaming a referenced command therefore breaks the CLI
contract test rather than leaving stale executable prose.

## Token-size tradeoffs

The shared contract has a deterministic golden hash and a 7,000-byte guardrail.
Keep safety and stop conditions inline because every runtime must receive them;
put explanatory tutorials in documentation or one-level skill resources. A
workspace override that adds tokens is paid on every invocation and every
supervise turn, so compare the resolved size and measured run cost before
expanding it. Do not shorten the prompt by deleting a required typed section:
the registry test rejects missing sections and role lifecycles.

MCP descriptions are currently embedded-only — the server builds its tool list before it has a workspace. Known limitation, not a decision.

## The boundary, so it doesn't erode

**One-line refusal and usage messages stay in code.** They are the exit-code contract's surface: tested by exact string, versioned with the behavior they describe, and meaningless apart from it. The rule of thumb: if it teaches an agent how to behave, it's a prompt (file); if it reports what just happened, it's a message (code). The 49 `refusedf`/`usagef` one-liners are messages.

## Audit trail

Extracted 2026-07-21 from five sites: `protocolPreamble`'s nine-Fprintf chain (the complaint that triggered this), the supervise correction, the brief header, the MCP refusal text, and 16 inline tool-description literals. The extraction immediately paid once: `get_context`'s description had already drifted (it didn't mention the Lessons section added by P1) — fixed in the registry file, where the next drift will be a one-line diff instead of an archaeology dig.
