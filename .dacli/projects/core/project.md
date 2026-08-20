---
id: p-core
kind: project
created: 2026-07-21T14:31:03Z
created_by: a-root
status: active
stage: approach
github_repo: mlnomadpy/dacli
github_public_confirmed: mlnomadpy/dacli
landing.mode: pr
landing.base: main
---
# dacli remaining backlog
## Goal
Keep dacli a provider-neutral, governed controller for coding-agent CLIs whose
durable record agrees with executed work, whose cost-aware routing selects the
cheapest capable model, and whose bounded loops land verified changes through
GitHub without implying publication authority.
## Constraints
## Out of scope
PUBLISHING — the act, not the machinery. Do not push a `v*` tag, do not create
or configure a Homebrew tap, and do not upload artifacts anywhere. Taha decides
when a version is solid enough to publish and will say so explicitly.

Release ENGINEERING is now in scope (2026-08-10, reversing the earlier
blanket ban, per issue #437): goreleaser config, checksums, SBOMs,
cross-platform binaries, and verifying the whole path with a SNAPSHOT build are
all work to do. The line is the tag push: everything up to it is engineering,
the tag itself is publication.

`.github/workflows/release.yml` fires only on a manually pushed `v*` tag and
never creates one. Keep it that way.

## Success criteria
- Every shipped CLI/MCP contract is executable, tested, and documented against the same command registry.
- Codex, Claude Code, Gemini, Copilot, and generic adapters share provider-neutral workflow semantics while preserving provider-specific transport and context controls.
- Task, wave, loop, and future service operation remain bounded, recoverable, budgeted, and explicit about landing and release authority.
- Acceptance and GitHub closure follow verified landed-main state; recovery artifacts preserve attribution instead of rewriting history.
- The open backlog contains evidence-backed, estimated, checkable tasks ordered by dependencies and critical-path impact.
## Codebase map
Snapshot measured from repository-visible source files on 2026-08-19; task 467
adds source-controlled drift checks for normative architecture and inventory.

**Languages:**
- Go (340 files; 91,156 lines)
- Markdown (72 files)
- JavaScript (0 files)
- TypeScript (34 files)
- Shell (5 files)

**Top-level structure:**
- cmd/
- docs/
- internal/
- overrides/
- scripts/
- site/

**Existing docs:**
- CHANGELOG.md
- DESIGN.md
- README.md
- SYSTEM_AUDIT_2026-07-27.md
- docs/ARCHITECTURE.md
- docs/COMPATIBILITY.md
- docs/DIAGRAMS.md
- docs/DOGFOOD.md
- docs/FORMAT.md
- docs/GITHUB.md
- docs/GITHUB_APP.md
- docs/MCP.md
- docs/MULTI_CLI.md
- docs/OPERATOR_PLAYBOOK.md
- docs/PROMPTS.md
- docs/PROPOSALS.md
- docs/README.md
- docs/REVIEW.md
- docs/ROSTER.md
- docs/RUNTIMES.md
- docs/SELFHOSTING.md
- docs/SHORTCUTS.md
- docs/SKILLS.md
- docs/SPM.md
- docs/TEAM.md
- docs/TEMPLATES.md
- docs/TRUST.md
- docs/WALKTHROUGH.md
- docs/index.md
- docs/research/DASHBOARD_UX_RESEARCH.md
- docs/research/INTERVIEW_GUIDE.md
