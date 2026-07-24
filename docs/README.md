# dacli documentation

Reading order top to bottom. **Status** says how real each document is — a spec that pretends to be implemented is worse than either.

| Doc | What it covers | Status |
|---|---|---|
| [DESIGN.md](https://github.com/mlnomadpy/dacli/blob/main/DESIGN.md) | Problem, object model, permissions, concurrency, non-goals | Contract (draft) |
| [ARCHITECTURE.md](ARCHITECTURE.md) | Axioms, layers, build order, interface contracts, the canonical brief | **Normative** — wins on overlap |
| [FORMAT.md](FORMAT.md) | Every file on disk, field by field | Stable-intent, `format: 0` |
| [SPM.md](SPM.md) | Which management frameworks port to agents, and which deliberately don't | **Implemented** |
| [TEAM.md](TEAM.md) | Roles, scope, escalation-not-chat, spawning, model tiering | **Implemented** |
| [SHORTCUTS.md](SHORTCUTS.md) | Memoized commands: quoting, effects, promotion | **Implemented** |
| [RUNTIMES.md](RUNTIMES.md) | Driving coding-agent CLIs; supervision; verify panels | **Implemented** |
| [SKILLS.md](SKILLS.md) | One canonical skill format, compiled to each CLI's delivery mechanism | **Implemented** |
| [MCP.md](MCP.md) | The agent-primary interface: tiered tools, refusals-as-results | **Implemented** (`dacli mcp serve`) |
| [PROMPTS.md](PROMPTS.md) | Every agent-facing prompt as a reviewable, overridable file | **Implemented** |
| [TEMPLATES.md](TEMPLATES.md) | Project kinds, required docs, stage gates | **Implemented** (v1) |
| [GITHUB.md](GITHUB.md) | Issues/Projects as a projection; humans as events; Obsidian | **Implemented** |
| [WALKTHROUGH.md](WALKTHROUGH.md) | One task traced end to end through everything above; § 9 zooms out to the perpetual loop, the governor, and landing | Illustrative (§ 9 implemented) |
| [ROSTER.md](ROSTER.md) | Generated team roster: every role and skill, one table each | Generated (`dacli catalog`) |
| [PROPOSALS.md](PROPOSALS.md) | The four learning loops (all shipped), tier 2, and the rejections | **All four loops shipped** |
| [REVIEW.md](REVIEW.md) | The 2026-07-21 full-design audit: defects, gaps, dispositions | Record |
| [research/INTERVIEW_GUIDE.md](research/INTERVIEW_GUIDE.md) | Discovery research plan + interview scripts for dashboard/steering features, by segment | Research instrument |
| [research/DASHBOARD_UX_RESEARCH.md](research/DASHBOARD_UX_RESEARCH.md) | Synthesis of the four segment interviews: personas, human-vs-agent needs matrix, key tensions, RICE roadmap + ready-to-build shortlist | Research synthesis |

The whole spec is now built: no `planned()` stubs remain in product code. See [ARCHITECTURE.md § 2b](ARCHITECTURE.md) for the feature-sliced layout.
