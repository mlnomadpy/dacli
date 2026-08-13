---
id: t-01KZYQ5E5ZAKQDMM2DF4AGQ8VQ
kind: task
created: 2026-08-13T23:26:21Z
created_by: a-root
owner: a-root
github:
  issue: 650
  repo: mlnomadpy/dacli
---
# Upgrade the dacli Codex skill into a complete operator playbook
## Context
Adopted from GitHub issue #650.

## Problem

`skills/dacli/SKILL.md` contains valuable operational knowledge, but it is a 434-line monolith and omits or under-explains several behaviors an agent must understand before operating dacli safely: project scoping versus workspace-global task resolution, provider-neutral runtime selection across Codex/Claude/Gemini/Copilot/generic CLIs, GitHub-first collaboration, role/runtime/grant compatibility, and recovery paths for governed loops.

The result is expensive rediscovery and a risk that agents interpret projects as security boundaries, route a write task to a read-only runtime, bypass dacli landing records with raw GitHub operations, or run loops without the right bounds and finalization steps.

## Design

Use progressive disclosure:

- Keep `SKILL.md` as the concise mandatory workflow and safety contract.
- Add one-level `references/` guides for workspace/projects/tasks, runtimes/model routing, waves/loops/recovery, and GitHub/landing.
- State which reference to read for each class of request.
- Derive commands and behavioral claims from the current CLI/help and repository documentation.

## Acceptance criteria

- [ ] `skills/dacli/SKILL.md` has valid two-field skill frontmatter and stays below 500 lines.
- [ ] The core workflow tells an agent how to inspect state, select or create a project-scoped task, form checkable acceptance criteria, estimate/route it, claim paths, spawn in worktrees, wait/sync, verify, integrate, and push through GitHub.
- [ ] The skill explicitly explains that projects are organizational/scheduling scopes rather than access-control boundaries; direct task refs are workspace-global and ambiguous short refs must not be guessed.
- [ ] The runtime guidance covers Codex, Claude Code, Gemini CLI, GitHub Copilot CLI, and generic executors without making one provider the framework default.
- [ ] Model guidance explains provider-neutral role profiles, capacity/cost routing, task complexity versus blast radius, runtime/grant compatibility, and token controls.
- [ ] Loop/swarm guidance covers dry-run, bounds, width, path claims, worktrees, `wait`, `sync`, stop latch, no-progress recovery, and independent verification.
- [ ] GitHub guidance preserves dacli as the record owner while using issues/PRs/checks as the shared collaboration surface; external effects are previewed first.
- [ ] Recovery guidance maps exit codes and common failure states to non-destructive next actions.
- [ ] Detailed material is split into directly linked one-level reference files with no duplicated source of truth.
- [ ] The installed local copy is refreshed from the repository source and the skill passes the official `quick_validate.py` validator.

## Verification

- Run the official skill validator.
- Compare every documented command family against `go run ./cmd/dacli --help`.
- Check every Markdown link from `SKILL.md` resolves.
- Confirm the installed copy matches `skills/dacli/`.
- Review the diff for unsupported provider-specific assumptions and duplicated guidance.

## Acceptance
- [x] `skills/dacli/SKILL.md` has valid two-field skill frontmatter and stays below 500 lines.
- [x] The core workflow tells an agent how to inspect state, select or create a project-scoped task, form checkable acceptance criteria, estimate/route it, claim paths, spawn in worktrees, wait/sync, verify, integrate, and push through GitHub.
- [x] The skill explicitly explains that projects are organizational/scheduling scopes rather than access-control boundaries; direct task refs are workspace-global and ambiguous short refs must not be guessed.
- [x] The runtime guidance covers Codex, Claude Code, Gemini CLI, GitHub Copilot CLI, and generic executors without making one provider the framework default.
- [x] Model guidance explains provider-neutral role profiles, capacity/cost routing, task complexity versus blast radius, runtime/grant compatibility, and token controls.
- [x] Loop/swarm guidance covers dry-run, bounds, width, path claims, worktrees, `wait`, `sync`, stop latch, no-progress recovery, and independent verification.
- [x] GitHub guidance preserves dacli as the record owner while using issues/PRs/checks as the shared collaboration surface; external effects are previewed first.
- [x] Recovery guidance maps exit codes and common failure states to non-destructive next actions.
- [x] Detailed material is split into directly linked one-level reference files with no duplicated source of truth.
- [x] The installed local copy is refreshed from the repository source and the skill passes the official `quick_validate.py` validator.
## Log
- 2026-08-13T23:26:28Z claimed by a-root
- 2026-08-13T23:48:24Z accepted by a-root
- 2026-08-13T23:48:24Z verified by `python3 /Users/tahabsn/.codex/skills/.system/skill-creator/scripts/quick_validate.py skills/dacli && diff -qr skills/dacli /Users/tahabsn/.codex/skills/dacli` (exit 0) in branch main at a70eced — proves that tree builds, not that the work is in trunk
- 2026-08-13T23:48:24Z deliverable: dacli/440-upgrade-the-dacli-codex-skill-into-a-complete-operator-playbook is merged into main
- 2026-08-13T23:48:24Z completed by a-root
## Verification Evidence
{"command":"python3 /Users/tahabsn/.codex/skills/.system/skill-creator/scripts/quick_validate.py skills/dacli \u0026\u0026 diff -qr skills/dacli /Users/tahabsn/.codex/skills/dacli","exit_code":0,"duration_ms":35,"artifact_hash":"sha256:db349825903d66adffea3ecf1bd8e1803043e8a71cf1a051235dabc5371f5bb0","verifier":"a-root","branch":"codex/650-upgrade-dacli-skill","commit_sha":"9942ad30813e7361e2657453ce1a29e822bc4602"}
{"command":"python3 /Users/tahabsn/.codex/skills/.system/skill-creator/scripts/quick_validate.py skills/dacli \u0026\u0026 diff -qr skills/dacli /Users/tahabsn/.codex/skills/dacli","exit_code":0,"duration_ms":115,"artifact_hash":"sha256:db349825903d66adffea3ecf1bd8e1803043e8a71cf1a051235dabc5371f5bb0","verifier":"a-root","branch":"main","commit_sha":"a70ecedf689b5732e13eafcbd7d87b63e4dc2bed"}
