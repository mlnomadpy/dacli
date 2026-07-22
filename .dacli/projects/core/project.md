---
id: p-core
kind: project
created: 2026-07-21T14:31:03Z
created_by: a-root
status: active
stage: approach
---
# dacli remaining backlog
## Goal
Every planned() stub implemented, honestly, against its spec doc.
## Constraints
## Out of scope
## Success criteria
## Codebase map
**Languages:**
- Go (83 files)
- Markdown (30 files)

**Top-level structure:**
- cmd/
- docs/
- internal/

**Existing docs:**
- DESIGN.md
- README.md
- docs/ARCHITECTURE.md
- docs/FORMAT.md
- docs/GITHUB.md
- docs/MCP.md
- docs/PROMPTS.md
- docs/PROPOSALS.md
- docs/README.md
- docs/REVIEW.md
- docs/RUNTIMES.md
- docs/SELFHOSTING.md
- docs/SHORTCUTS.md
- docs/SKILLS.md
- docs/SPM.md
- docs/TEAM.md
- docs/TEMPLATES.md
- docs/WALKTHROUGH.md

**Open markers (13):**
- TODO internal/cli/onboard_test.go:16 — handle the batch path\nfunc Pay() {}\n")
- FIXME internal/cli/onboard_test.go:17 — flaky under -race\n")
- XXX internal/cli/onboard_test.go:18 — no error handling\nexport const app = 1\n")
- TODO internal/cli/onboard_test.go:52 — not turned into a task:\n%s", tasks)
- TODO internal/cli/onboard_test.go:55 — marker") {
- TODO internal/features/onboard/onboard.go:3 — markers the
- TODO internal/features/onboard/onboard.go:30 — tasks", Run: cmdAdopt},
- TODO internal/features/onboard/onboard.go:108 — tasks (%d more found) — the rest are in the codebase map\n", len(scan.todos)-created)
- TODO internal/features/onboard/onboard.go:122 — /FIXME markers\n", created)
- TODO internal/features/onboard/onboard.go:124 — /FIXME markers found — re-run with --todos to seed them as tasks\n", len(scan.todos))
- TODO internal/features/onboard/onboard.go:173 — markers — cheap scan of text-ish files only.
- TODO internal/features/onboard/onboard.go:198 — ", "FIXME", "HACK", "XXX"} {
- TODO internal/gates/gates.go:448 — ", "FIXME", "{{", "..."} {
