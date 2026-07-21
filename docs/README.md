# dacli documentation

Reading order top to bottom. **Status** says how real each document is — a spec that pretends to be implemented is worse than either.

| Doc | What it covers | Status |
|---|---|---|
| [../DESIGN.md](../DESIGN.md) | Problem, object model, permissions, concurrency, non-goals | Contract (draft) |
| [ARCHITECTURE.md](ARCHITECTURE.md) | Axioms, layers, build order, interface contracts, the canonical brief | **Normative** — wins on overlap |
| [FORMAT.md](FORMAT.md) | Every file on disk, field by field | Stable-intent, `format: 0` |
| [SPM.md](SPM.md) | Which management frameworks port to agents, and which deliberately don't | Engines implemented |
| [TEAM.md](TEAM.md) | Roles, scope, escalation-not-chat, spawning | Engines implemented |
| [SHORTCUTS.md](SHORTCUTS.md) | Memoized commands: quoting, effects, promotion | Engine implemented |
| [RUNTIMES.md](RUNTIMES.md) | Driving coding-agent CLIs; supervision; enforcement | Spec only |
| [SKILLS.md](SKILLS.md) | One canonical skill format, compiled to each CLI's delivery mechanism | Spec only |
| [MCP.md](MCP.md) | The agent-primary interface: tiered tools, refusals-as-results | Spec only |
| [TEMPLATES.md](TEMPLATES.md) | Project kinds, required docs, stage gates | Spec only |
| [GITHUB.md](GITHUB.md) | Issues/Projects as a projection; humans as events; Obsidian | Spec only |
| [WALKTHROUGH.md](WALKTHROUGH.md) | One task traced end to end through everything above | Illustrative |
| [PROPOSALS.md](PROPOSALS.md) | Ranked feature proposals: the four learning loops, tier 2, and the rejections | Proposals |
| [REVIEW.md](REVIEW.md) | The 2026-07-21 full-design audit: defects, gaps, dispositions | Record |

"Engines implemented" means the pure-computation layer (`internal/spm`, `internal/team`, `internal/shortcut`) is built and tested while every command consuming it is a stub — see [ARCHITECTURE.md § 2](ARCHITECTURE.md) for why the spine comes next.
