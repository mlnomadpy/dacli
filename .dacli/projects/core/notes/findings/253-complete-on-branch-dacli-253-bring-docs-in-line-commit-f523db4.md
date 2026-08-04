---
id: f-253-complete-on-branch-dacli-253-bring-docs-in-line-commit-f523db4
kind: note
note_kind: finding
created: 2026-08-04T11:20:18Z
created_by: a-maintainer-vgqd2d
about: "[[253]]"
severity: moderate
---
# 253 complete on branch dacli/253-bring-docs-in-line-... (commit f523db4)
All 4 acceptance criteria met. (1) docs/GITHUB.md §9.5 now documents dacli integrate as THE merge path (not hand git-merge) and the dacli/<seq>-<slug> branch key (fmt.Sprintf %03d-slug, internal/features/vcs/lifecycle.go:41), incl. worktree keyed on project+seq+slug (288). (2) docs/TEAM.md §4 states grant+runtime must agree — ro grant on a runtime with empty SandboxRO is refused exit 3 (sandboxFor, execution.go:1017-1029), --cooperative warns instead — and how to check: dacli runtime doctor/list shows read-only status; docs/ROSTER.md carries the same note via the generated preamble in catalog.go (persists across dacli catalog regen). (3) docs/SELFHOSTING.md fixed the wrong ff-only merge flow (git_workflow.md never taught it) and states the real 96 merged PRs (#39-#293, git log main); index.md/README.md/overrides/home.html updated from stale 77->96. (4) docs/README.md index adds SELFHOSTING.md + DOGFOOD.md rows and a note covering index.md/README.md, so every docs/ file is accounted for. Proof: go build ./... clean; go vet ./... clean; gofmt -l internal/ clean; go test ./... green with DACLI_AGENT stripped (the known env-leak isolation gap, not a regression); new catalog_test.go assertion fails before the change (verified via git stash) and passes after. PR-first off: owner to accept 253 + integrate --tasks 253 --into main.
