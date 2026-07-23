---
id: t-01KY849P476FYS4MTCTASYC0XW
kind: task
created: 2026-07-23T18:37:38Z
created_by: a-root
owner: a-root
github:
  issue: 64
  repo: mlnomadpy/dacli
---
# [agent-report] dacli adopt without --project silently creates a NEW project instead of refreshing the existing one, and there is no project delete command to recover
## Context
Adopted from GitHub issue #64.

Repro: workspace already has project 'core' (repo dir is named 'dacli', not 'core'). Ran 'dacli adopt' with no --project flag while trying to refresh core's codebase map. cmdAdopt (internal/features/onboard/onboard.go:70-73) defaults slug to store.Slugify(filepath.Base(w.Root)) when --project is empty, which resolved to 'dacli', a DIFFERENT slug than 'core'. store.LoadProject(w, 'dacli') failed (not found) so cmdAdopt took the CreateProject branch and silently created a brand-new project 'dacli' alongside 'core', with its own freshly-scanned codebase map — instead of erring, warning, or refreshing the one existing project. There is also no 'dacli project rm/delete' command, so recovering from the mistake required manually os.RemoveAll'ing the project's directory under .dacli/projects/<slug>/ via a throwaway Go program (git clean -fd could not reach it since the file lived in a different git worktree than the one running the fix). Suggest: when exactly one project exists and --project is omitted, default to IT rather than a derived-from-dirname slug; and/or add a 'dacli project rm <slug>' command for exactly this kind of recovery.

---
_Reported via `dacli report`._
- dacli: dev
- platform: darwin/arm64
- workspace and run transcript withheld (public upstream) — re-run with --disclose to include them

## Acceptance
## Log
