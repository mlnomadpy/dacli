# dacli documentation

Reading order top to bottom. **Status** says how real each document is — a spec that pretends to be implemented is worse than either.

| Doc | What it covers | Status |
|---|---|---|
| [../DESIGN.md](../DESIGN.md) | Problem, object model, permissions, concurrency, non-goals | Contract (draft) |
| [ARCHITECTURE.md](ARCHITECTURE.md) | Axioms, layers, build order, interface contracts, the canonical brief | **Normative** — wins on overlap |
| [FORMAT.md](FORMAT.md) | Every file on disk, field by field | Stable-intent, `format: 0` |
| [SPM.md](SPM.md) | Which management frameworks port to agents, and which deliberately don't | **Implemented** |
| [TEAM.md](TEAM.md) | Roles, scope, escalation-not-chat, spawning, model tiering | **Implemented** |
| [SHORTCUTS.md](SHORTCUTS.md) | Memoized commands: quoting, effects, promotion | **Implemented** (promote planned) |
| [RUNTIMES.md](RUNTIMES.md) | Driving coding-agent CLIs; supervision; verify panels | **Implemented** |
| [SKILLS.md](SKILLS.md) | One canonical skill format, compiled to each CLI's delivery mechanism | **Implemented** (promote planned) |
| [MCP.md](MCP.md) | The agent-primary interface: tiered tools, refusals-as-results | **Implemented** (`dacli mcp serve`) |
| [PROMPTS.md](PROMPTS.md) | Every agent-facing prompt as a reviewable, overridable file | **Implemented** |
| [TEMPLATES.md](TEMPLATES.md) | Project kinds, required docs, stage gates | **Implemented** (v1) |
| [GITHUB.md](GITHUB.md) | Issues/Projects as a projection; humans as events; Obsidian | **Implemented** (outbound; inbound planned) |
| [WALKTHROUGH.md](WALKTHROUGH.md) | One task traced end to end through everything above | Illustrative |
| [PROPOSALS.md](PROPOSALS.md) | The four learning loops (all shipped), tier 2, and the rejections | **All four loops shipped** |
| [REVIEW.md](REVIEW.md) | The 2026-07-21 full-design audit: defects, gaps, dispositions | Record |

Nearly the whole spec is now built. Still genuinely unimplemented (honest `planned()` stubs that name their blocker): GitHub **inbound** sync (`github pull`/`sync`), `skill promote` and `shortcut promote` (both wait on an upstream: promotable lessons / ad-hoc command tracking). See [ARCHITECTURE.md § 2b](ARCHITECTURE.md) for the feature-sliced layout.
