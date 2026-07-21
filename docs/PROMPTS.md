# The prompt registry

**Status: implemented** (`internal/prompts`, `dacli prompt list|show`).

Every multi-sentence piece of agent-facing prose dacli emits lives in one place: template files under `internal/prompts/tpl/`, embedded at build time, overridable per workspace. This exists because a prompt buried in an `Fprintf` chain cannot be audited, diffed in a PR, or improved without recompiling — and for this tool, **prompts are load-bearing artifacts**. It is the same doctrine as adapters and shortcuts: prose is data, not code.

## The registry

| Prompt | Used by | Carries |
|---|---|---|
| `protocol_preamble` | `spawn`, `supervise` | How a child reports: the binary path, report-immediately, ask-don't-guess, never-retry-refused, and the rw/ro verb split |
| `supervise_correction` | `supervise` turns > 1 | The unmet criteria, named exactly |
| `brief_header` | every brief | The est-tokens line and the **data-not-instructions warning** — a security posture that deserves review as a file |
| `refusal_next` | MCP exit-3 mapping | The do-not-retry instruction attached to every refusal |
| `mcp_tools` | `mcp serve` | All 16 tool descriptions, one sectioned file — the entire agent-facing tool manual in a single reviewable diff |
| `git_workflow` | `spawn`/`supervise`, rw children | Branch-per-task (`dacli/NNN-slug`), commit discipline, red-suite-means-unchecked; the push-and-`gh pr create` flow only with `--pr`, otherwise an explicit do-not-push |
| `review_workflow` | `spawn`/`supervise` with `--review` | Judge the `gh pr diff` against acceptance criteria, not taste; every defect filed twice (dacli finding + PR comment); approve/request-changes semantics |

`dacli prompt list` shows the registry with overrides marked; `dacli prompt show <name>` prints the resolved template.

## Override rule

A file named `.dacli/prompts/<name>.md` wins over the embedded default — nearest-wins, same as templates. Prompt tuning becomes a **workspace commit**: attributable, revertible, reviewable, and visible to `dacli taint` like any other content. A broken `protocol_preamble` override fails the spawn (a child working into the void is worse than no child); a broken `brief_header` override falls back to the embedded default (a brief without the untrusted-content warning must not ship).

MCP descriptions are currently embedded-only — the server builds its tool list before it has a workspace. Known limitation, not a decision.

## The boundary, so it doesn't erode

**One-line refusal and usage messages stay in code.** They are the exit-code contract's surface: tested by exact string, versioned with the behavior they describe, and meaningless apart from it. The rule of thumb: if it teaches an agent how to behave, it's a prompt (file); if it reports what just happened, it's a message (code). The 49 `refusedf`/`usagef` one-liners are messages.

## Audit trail

Extracted 2026-07-21 from five sites: `protocolPreamble`'s nine-Fprintf chain (the complaint that triggered this), the supervise correction, the brief header, the MCP refusal text, and 16 inline tool-description literals. The extraction immediately paid once: `get_context`'s description had already drifted (it didn't mention the Lessons section added by P1) — fixed in the registry file, where the next drift will be a one-line diff instead of an archaeology dig.
