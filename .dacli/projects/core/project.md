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
Every planned() stub implemented, honestly, against its spec doc.
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
## Codebase map
**Languages:**
- Go (141 files)
- Markdown (43 files)
- JavaScript (36 files)
- TypeScript (33 files)
- Shell (1 files)

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
- docs/DOGFOOD.md
- docs/FORMAT.md
- docs/GITHUB.md
- docs/MCP.md
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
- docs/WALKTHROUGH.md
- docs/index.md
- docs/research/DASHBOARD_UX_RESEARCH.md
- docs/research/INTERVIEW_GUIDE.md
